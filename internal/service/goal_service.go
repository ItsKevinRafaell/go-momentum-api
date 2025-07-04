package service

import (
	"context"
	"errors"

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
	// 4. Panggil AI service yang asli, bukan mock lagi
	steps, err := s.aiService.GenerateRoadmapWithAI(ctx, goalDescription)
	if err != nil {
		return nil, nil, err
	}

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

	for i := range steps {
		steps[i].GoalID = goalID
	}

	err = s.roadmapRepo.CreateRoadmapSteps(ctx, steps)
	if err != nil {
		return nil, nil, err
	}

	return newGoal, steps, nil
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

// UpdateGoal mengorkestrasi proses update tujuan dan regenerasi roadmap.
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
    }
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
	// Di masa depan, kita bisa menambahkan logika di sini,
	// misalnya: jika semua step selesai, update status goal utama.
	return s.roadmapRepo.UpdateStepStatus(ctx, userID, stepID, status)
}