package service

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/ItsKevinRafaell/go-momentum-api/internal/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type GoalService struct {
	db *pgxpool.Pool
	goalRepo    *repository.GoalRepository
	roadmapRepo *repository.RoadmapRepository
	aiService   *AIService // <-- 1. Tambahkan dependensi ke AI Service
}

// 2. Terima AIService sebagai argumen
func NewGoalService(db *pgxpool.Pool, goalRepo *repository.GoalRepository, roadmapRepo *repository.RoadmapRepository, aiService *AIService) *GoalService {
    return &GoalService{
        db:          db,
        goalRepo:    goalRepo,
        roadmapRepo: roadmapRepo,
        aiService:   aiService,
    }
}

// Fungsi callAIToGenerateRoadmap yang lama bisa dihapus.
func (s *GoalService) CreateNewGoal(ctx context.Context, userID string, goalDescription string) (*repository.Goal, []repository.RoadmapStep, error) {
	// 1. Panggil AI untuk membuat roadmap
	stepsFromAI, err := s.aiService.GenerateRoadmapWithAI(ctx, goalDescription)
	if err != nil {
		return nil, nil, err
	}
	if len(stepsFromAI) == 0 {
		return nil, nil, errors.New("AI did not generate any roadmap steps")
	}

	// 2. Buat objek goal baru dan simpan
	newGoal := &repository.Goal{
		UserID:      userID,
		Description: goalDescription,
		IsActive:    true,
	}
	goalID, err := s.goalRepo.CreateGoal(ctx, newGoal)
	if err != nil {
		return nil, nil, err
	}
	newGoal.ID = goalID

	// --- PERUBAHAN UTAMA: SIMPAN LANGKAH SATU PER SATU ---
	var savedSteps []repository.RoadmapStep
	for _, step := range stepsFromAI {
		step.GoalID = goalID
		step.Status = "pending"

		// Panggil fungsi CreateRoadmapStep (single) yang sudah kita punya
		createdStep, err := s.roadmapRepo.CreateRoadmapStep(ctx, &step)
		if err != nil {
			// Jika satu langkah gagal disimpan, hapus goal yang sudah terlanjur dibuat
			// agar data tetap konsisten.
			log.Printf("Gagal menyimpan step, melakukan rollback dengan menghapus goal: %s", goalID)
			s.goalRepo.DeleteGoalByID(ctx, goalID, userID) // Anda mungkin perlu membuat fungsi ini
			return nil, nil, fmt.Errorf("gagal menyimpan langkah roadmap: %w", err)
		}
		savedSteps = append(savedSteps, *createdStep)
	}
	
	log.Printf("Berhasil menyimpan %d langkah roadmap baru.", len(savedSteps))
	return newGoal, savedSteps, nil
}

func (s *GoalService) GetActiveGoal(ctx context.Context, userID string) (*repository.Goal, []repository.RoadmapStep, error) {
	// 1. Dapatkan goal yang aktif
	goal, err := s.goalRepo.GetActiveGoalByUserID(ctx, userID)
	if err != nil {
		// Jika errornya adalah "tidak ada baris", berarti user belum punya goal.
		if err == pgx.ErrNoRows {
			return nil, nil, nil // Tidak dianggap error, hanya datanya tidak ada.
		}
		return nil, nil, err // Error lain yang tidak terduga
	}

	// 2. Jika goal ditemukan, dapatkan semua roadmap steps-nya
	steps, err := s.roadmapRepo.GetRoadmapStepsByGoalID(ctx, goal.ID)
	if err != nil {
		return nil, nil, err
	}

	return goal, steps, nil
}

// file: internal/service/goal_service.go

func (s *GoalService) UpdateGoal(ctx context.Context, userID, goalID, newDescription string) (*repository.Goal, []repository.RoadmapStep, error) {
    // 1. Update deskripsi goal di database
    if err := s.goalRepo.UpdateGoalDescription(ctx, userID, goalID, newDescription); err != nil {
        return nil, nil, err
    }

    // 2. Hapus semua roadmap steps yang lama
    if err := s.roadmapRepo.DeleteRoadmapStepsByGoalID(ctx, goalID); err != nil {
        return nil, nil, err
    }

    // 3. Panggil AI untuk membuat roadmap steps yang baru
    newSteps, err := s.aiService.GenerateRoadmapWithAI(ctx, newDescription)
    if err != nil {
        return nil, nil, err
    }

    // 4. Hubungkan dan simpan roadmap steps yang baru
    for i := range newSteps {
        newSteps[i].GoalID = goalID
        newSteps[i].Status = "pending" // <-- INI PERBAIKANNYA
    }
    // Gunakan CreateRoadmapSteps yang melakukan bulk insert
    if err := s.roadmapRepo.CreateRoadmapSteps(ctx, newSteps); err != nil {
        return nil, nil, err
    }

    // 5. Ambil data goal yang sudah terupdate untuk dikembalikan
    updatedGoal, err := s.goalRepo.GetActiveGoalByUserID(ctx, userID)
    if err != nil {
        return nil, nil, err
    }

    return updatedGoal, newSteps, nil
}

func (s *GoalService) AddRoadmapStep(ctx context.Context, goalID, title string) (*repository.RoadmapStep, error) {
    // 1. Dapatkan urutan terakhir
    lastOrder, err := s.roadmapRepo.GetLastStepOrder(ctx, goalID)
    if err != nil {
        return nil, err
    }

    // 2. Buat objek step baru dengan urutan + 1
    newStep := &repository.RoadmapStep{
        GoalID:  goalID,
        Order:   lastOrder + 1,
        Title:   title,
        Status:  "pending",
    }

    // 3. Simpan ke database
    return s.roadmapRepo.CreateRoadmapStep(ctx, newStep)
}

func (s *GoalService) UpdateRoadmapStep(ctx context.Context, userID, stepID, newTitle string) error {
    return s.roadmapRepo.UpdateStepTitle(ctx, userID, stepID, newTitle)
}

func (s *GoalService) DeleteRoadmapStep(ctx context.Context, userID, stepID string) error {
    // Kita butuh transaksi karena ada beberapa operasi database
    tx, err := s.db.Begin(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback(ctx)

    // 1. Dapatkan detail step yang mau dihapus untuk tahu order & goalId-nya
    stepToDelete, err := s.roadmapRepo.GetStepByID(ctx, stepID)
    if err != nil {
        return errors.New("step not found")
    }

    // 2. Validasi kepemilikan
    activeGoal, err := s.goalRepo.GetActiveGoalByUserID(ctx, userID)
    if err != nil || activeGoal.ID != stepToDelete.GoalID {
        return errors.New("user does not have permission to delete this step")
    }

    // 3. Hapus step di dalam transaksi
    if err := s.roadmapRepo.DeleteRoadmapStep(ctx, tx, stepID); err != nil {
        return err
    }

    // 4. Perbarui urutan step lain di dalam transaksi yang sama
    if err := s.roadmapRepo.RenumberStepsAfterDelete(ctx, tx, stepToDelete.GoalID, stepToDelete.Order); err != nil {
        return err
    }

    // 5. Jika semua berhasil, commit transaksinya
    return tx.Commit(ctx)
}
func (s *GoalService) ReorderRoadmapSteps(ctx context.Context, userID string, stepIDs []string) error {
	return s.roadmapRepo.ReorderRoadmapSteps(ctx, userID, stepIDs)
}

func (s *GoalService) UpdateRoadmapStepStatus(ctx context.Context, userID, stepID, status string) error {
	// Untuk saat ini, service hanya meneruskan panggilan ke repository yang sudah aman
	return s.roadmapRepo.UpdateStepStatus(ctx, userID, stepID, status)
}