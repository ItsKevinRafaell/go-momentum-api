// file: internal/service/task_service.go
package service

import (
	"context"
	"log"
	"time"

	"github.com/ItsKevinRafaell/go-momentum-api/internal/repository"
)

type TaskService struct {
	taskRepo    *repository.TaskRepository
	goalRepo    *repository.GoalRepository
	roadmapRepo *repository.RoadmapRepository
	aiService   *AIService
}

func NewTaskService(taskRepo *repository.TaskRepository, goalRepo *repository.GoalRepository, roadmapRepo *repository.RoadmapRepository, aiService *AIService) *TaskService {
	return &TaskService{
		taskRepo:    taskRepo,
		goalRepo:    goalRepo,
		roadmapRepo: roadmapRepo,
		aiService:   aiService,
	}
}

// GetOrCreateTodaySchedule adalah fungsi utama kita, sekarang dengan alur yang benar.
func (s *TaskService) GetOrCreateTodaySchedule(ctx context.Context, userID string) ([]repository.Task, error) {
	// Untuk konsistensi, gunakan awal hari sebagai patokan tanggal
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

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

	activeGoal, err := s.goalRepo.GetActiveGoalByUserID(ctx, userID)
	if err != nil || activeGoal == nil {
		// Jika tidak ada goal aktif, kembalikan array kosong, bukan error
		return []repository.Task{}, nil
	}

	yesterday := today.AddDate(0, 0, -1)
	yesterdayTasks, err := s.taskRepo.GetTasksByDate(ctx, userID, yesterday)
	if err != nil {
		return nil, err
	}

	// Panggil method dari aiService yang asli
	newTasksFromAI, err := s.aiService.GenerateDailyTasksWithAI(ctx, activeGoal.Description, yesterdayTasks)
	if err != nil {
		return nil, err
	}

	// Jika AI tidak menghasilkan tugas, kembalikan array kosong
	if len(newTasksFromAI) == 0 {
		return []repository.Task{}, nil
	}

	// Simpan tugas satu per satu untuk mendapatkan ID unik dari database
	var createdTasks []repository.Task
	for _, taskToCreate := range newTasksFromAI {
		taskToCreate.UserID = userID
		taskToCreate.Status = "pending"
		taskToCreate.ScheduledDate = today

		createdTask, err := s.taskRepo.CreateTask(ctx, &taskToCreate)
		if err != nil {
			return nil, err // Jika satu gagal, hentikan proses
		}
		createdTasks = append(createdTasks, *createdTask)
	}

	log.Printf("Berhasil membuat %d tugas baru dari AI untuk hari ini.", len(createdTasks))
	return createdTasks, nil
}

// FinalizeDayReview menjalankan proses akhir hari.
func (s *TaskService) FinalizeDayReview(ctx context.Context, userID string) ([]repository.TaskSummary, string, error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	err := s.taskRepo.FinalizeMissedTasks(ctx, userID, today)
	if err != nil {
		return nil, "", err
	}
	summary, err := s.taskRepo.GetTaskSummaryByDate(ctx, userID, today)
	if err != nil {
		return nil, "", err
	}
	feedback, err := s.aiService.GenerateReviewFeedback(ctx, summary)
	if err != nil {
		feedback = "Gagal mendapatkan feedback dari AI, tapi tetap semangat untuk esok hari!"
	}
	return summary, feedback, nil
}

// --- FUNGSI-FUNGSI UNTUK MODIFIKASI TUGAS ---

func (s *TaskService) CreateManualTask(ctx context.Context, userID, title string, deadline *time.Time) (*repository.Task, error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	newTask := &repository.Task{
		UserID:        userID,
		Title:         title,
		Status:        "pending",
		ScheduledDate: today,
		Deadline:      deadline,
	}
	return s.taskRepo.CreateTask(ctx, newTask)
}

func (s *TaskService) UpdateTaskStatus(ctx context.Context, userID string, taskID string, status string) error {
	return s.taskRepo.UpdateTaskStatus(ctx, userID, taskID, status)
}

func (s *TaskService) UpdateTaskDeadline(ctx context.Context, userID, taskID string, deadline time.Time) error {
	return s.taskRepo.UpdateTaskDeadline(ctx, userID, taskID, deadline)
}

func (s *TaskService) UpdateTaskTitle(ctx context.Context, userID string, taskID string, title string) error {
	return s.taskRepo.UpdateTaskTitle(ctx, userID, taskID, title)
}

func (s *TaskService) DeleteTask(ctx context.Context, userID, taskID string) error {
	return s.taskRepo.DeleteTask(ctx, userID, taskID)
}