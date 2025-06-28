package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ItsKevinRafaell/go-momentum-api/internal/repository"
)

type TaskService struct {
	taskRepo    *repository.TaskRepository
	goalRepo    *repository.GoalRepository
	roadmapRepo *repository.RoadmapRepository
}

func NewTaskService(taskRepo *repository.TaskRepository, goalRepo *repository.GoalRepository, roadmapRepo *repository.RoadmapRepository) *TaskService {
	return &TaskService{
		taskRepo:    taskRepo,
		goalRepo:    goalRepo,
		roadmapRepo: roadmapRepo,
	}
}

// --- FUNGSI AI PALSU (MOCK) ---
// Fungsi ini mensimulasikan AI yang membuat jadwal harian.
func (s *TaskService) callAIToGenerateDailyTasks(ctx context.Context, userID string, goal *repository.Goal, yesterdayTasks []repository.Task) ([]repository.Task, error) {
	log.Printf("AI (palsu) membuat jadwal untuk user: %s", userID)
	log.Printf("Tujuan utama: %s", goal.Description)
	log.Printf("Jumlah tugas kemarin: %d", len(yesterdayTasks))

	// Logika AI palsu: Buat 3 tugas dummy untuk hari ini.
	today := time.Now()
	newTasks := []repository.Task{
		{UserID: userID, Title: "Tugas #1 dari AI untuk hari ini", Status: "pending", ScheduledDate: today},
		{UserID: userID, Title: "Tugas #2 dari AI (mungkin menjadwalkan ulang yang kemarin)", Status: "pending", ScheduledDate: today},
		{UserID: userID, Title: "Tugas #3 dari AI, berkaitan dengan tujuan", Status: "pending", ScheduledDate: today},
	}

	return newTasks, nil
}

// GetOrCreateTodaySchedule adalah fungsi utama kita.
func (s *TaskService) GetOrCreateTodaySchedule(ctx context.Context, userID string) ([]repository.Task, error) {
	today := time.Now()

	// 1. Cek apakah jadwal untuk hari ini sudah ada
	existingTasks, err := s.taskRepo.GetTasksByDate(ctx, userID, today)
	if err != nil {
		return nil, err
	}

	// 2. Jika sudah ada, langsung kembalikan
	if len(existingTasks) > 0 {
		log.Println("Jadwal hari ini sudah ada, mengembalikan dari database.")
		return existingTasks, nil
	}

	// 3. Jika belum ada, mulai proses generasi jadwal baru
	log.Println("Jadwal hari ini belum ada. Memulai proses generasi baru...")

	// 3a. Dapatkan konteks: goal aktif pengguna
	activeGoal, err := s.goalRepo.GetActiveGoalByUserID(ctx, userID)
	if err != nil {
		// Jika tidak ada goal aktif, tidak bisa membuat jadwal.
		return []repository.Task{}, nil
	}

	// 3b. Dapatkan konteks: tugas hari kemarin untuk review
	yesterday := today.AddDate(0, 0, -1)
	yesterdayTasks, err := s.taskRepo.GetTasksByDate(ctx, userID, yesterday)
	if err != nil {
		return nil, err
	}

	// 3c. Panggil AI dengan semua konteks yang ada
	newTasks, err := s.callAIToGenerateDailyTasks(ctx, userID, activeGoal, yesterdayTasks)
	if err != nil {
		return nil, err
	}

	// 3d. Simpan tugas-tugas baru ke database
	if len(newTasks) > 0 {
		err = s.taskRepo.CreateTasks(ctx, newTasks)
		if err != nil {
			return nil, err
		}
	}
    log.Printf("Berhasil membuat %d tugas baru untuk hari ini.", len(newTasks))

	return newTasks, nil
}

func (s *TaskService) UpdateTaskStatus(ctx context.Context, userID, taskID, status string) error {
    // Di sini kita bisa menambahkan validasi jika perlu,
    // misalnya memastikan statusnya hanya boleh "completed" atau "pending".
    // Untuk sekarang, kita langsung teruskan.
	return s.taskRepo.UpdateTaskStatus(ctx, userID, taskID, status)
}


func (s *TaskService) UpdateTaskDeadline(ctx context.Context, userID, taskID string, deadline time.Time) error {
	return s.taskRepo.UpdateTaskDeadline(ctx, userID, taskID, deadline)
}

func (s *TaskService) UpdateTaskTitle(ctx context.Context, userID, taskID, title string) error {
	return s.taskRepo.UpdateTaskTitle(ctx, userID, taskID, title)
}

// file: internal/service/task_service.go
// ... (kode yang sudah ada)

func (s *TaskService) CreateManualTask(ctx context.Context, userID, title string, deadline *time.Time) (*repository.Task, error) {
	newTask := &repository.Task{
		UserID:        userID,
		Title:         title,
		Status:        "pending",
		ScheduledDate: time.Now(),
		Deadline:      deadline, // Akan menjadi NULL di database jika `deadline` adalah nil
	}

	createdTask, err := s.taskRepo.CreateTask(ctx, newTask)
	if err != nil {
		return nil, err
	}

	return createdTask, nil
}

func (s *TaskService) DeleteTask(ctx context.Context, userID, taskID string) error {
	return s.taskRepo.DeleteTask(ctx, userID, taskID)
}

// --- FUNGSI AI PALSU (MOCK) UNTUK REVIEW ---
func (s *TaskService) callAIToGenerateReviewFeedback(ctx context.Context, summary []repository.TaskSummary) (string, error) {
	log.Println("AI (palsu) membuat feedback review...")

	// Kita buat map untuk menampung semua status agar lebih mudah diakses
	summaryMap := make(map[string]int)
	for _, s := range summary {
		summaryMap[s.Status] = s.Count
	}

	completed := summaryMap["completed"] // akan 0 jika tidak ada
	missed := summaryMap["missed"]       // akan 0 jika tidak ada
	pending := summaryMap["pending"]     // akan 0 jika tidak ada

	// Logika yang sudah diperbaiki:
	if completed > 0 && missed == 0 && pending == 0 {
		// Kasus 1: Hari yang sempurna, semua selesai.
		return "Kerja yang luar biasa hari ini! Semua target tercapai. Pertahankan momentum ini besok!", nil
	} else if completed > 0 {
		// Kasus 2: Ada progres, tapi masih ada sisa.
		return fmt.Sprintf("Progres yang bagus dengan %d tugas selesai! Jangan khawatir dengan yang tersisa, besok adalah hari yang baru.", completed), nil
	} else if missed > 0 {
        // Kasus 3: Tidak ada yang selesai, ada yang terlewat.
        return "Sepertinya hari ini cukup menantang. Tidak apa-apa, istirahatlah. Besok kita coba lagi!", nil
    } else {
		// Kasus 4: Tidak ada yang selesai, tapi tidak ada yang terlewat (semua masih pending).
		return "Setiap langkah berarti, bahkan istirahat sekalipun. Mari kita coba lagi dengan semangat baru besok!", nil
	}
}

// FinalizeDayReview menjalankan proses akhir hari.
func (s *TaskService) FinalizeDayReview(ctx context.Context, userID string) ([]repository.TaskSummary, string, error) {
	today := time.Now()

	// 1. Finalisasi tugas yang terlewat
	err := s.taskRepo.FinalizeMissedTasks(ctx, userID, today)
	if err != nil {
		return nil, "", err
	}

	// 2. Dapatkan ringkasan statistik setelah finalisasi
	summary, err := s.taskRepo.GetTaskSummaryByDate(ctx, userID, today)
	if err != nil {
		return nil, "", err
	}

	// 3. Panggil AI untuk mendapatkan feedback
	feedback, err := s.callAIToGenerateReviewFeedback(ctx, summary)
	if err != nil {
		return nil, "", err
	}

	return summary, feedback, nil
}