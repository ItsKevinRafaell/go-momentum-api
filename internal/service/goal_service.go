package service

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5"
	// Ganti dengan path modul Anda
	"github.com/ItsKevinRafaell/go-momentum-api/internal/repository"
)

type GoalService struct {
	goalRepo    *repository.GoalRepository
	roadmapRepo *repository.RoadmapRepository
}

func NewGoalService(goalRepo *repository.GoalRepository, roadmapRepo *repository.RoadmapRepository) *GoalService {
	return &GoalService{
		goalRepo:    goalRepo,
		roadmapRepo: roadmapRepo,
	}
}

// --- FUNGSI AI PALSU (MOCK) ---
// Nanti kita akan ganti ini dengan panggilan ke API Gemini yang sesungguhnya.
func (s *GoalService) callAIToGenerateRoadmap(ctx context.Context, goalDescription string) ([]repository.RoadmapStep, error) {
	log.Println("Memanggil AI (versi palsu) untuk membuat roadmap...")
	// Untuk sekarang, kita kembalikan data dummy/palsu.
	mockedSteps := []repository.RoadmapStep{
		{Order: 1, Title: "Minggu 1-2: Belajar fundamental dan setup lingkungan", Status: "pending"},
		{Order: 2, Title: "Minggu 3-4: Membuat proyek sederhana pertama", Status: "pending"},
		{Order: 3, Title: "Minggu 5: Belajar tentang testing dan deployment", Status: "pending"},
	}
	log.Println("AI (versi palsu) berhasil membuat roadmap.")
	return mockedSteps, nil
}

// CreateNewGoal adalah fungsi utama yang mengorkestrasi semuanya.
func (s *GoalService) CreateNewGoal(ctx context.Context, userID string, goalDescription string) (*repository.Goal, []repository.RoadmapStep, error) {
	// 1. Panggil AI untuk mendapatkan langkah-langkah roadmap
	steps, err := s.callAIToGenerateRoadmap(ctx, goalDescription)
	if err != nil {
		return nil, nil, err
	}

	// 2. Buat dan simpan Goal utama untuk mendapatkan ID-nya
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

	// 3. Tetapkan goalID untuk setiap langkah roadmap
	for i := range steps {
		steps[i].GoalID = goalID
	}

	// 4. Simpan semua langkah roadmap ke database
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