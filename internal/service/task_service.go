// file: internal/service/task_service.go
package service

import (
	"context"
	"log"
	"time"

	"github.com/ItsKevinRafaell/go-momentum-api/internal/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TaskService struct {
	db          *pgxpool.Pool
	taskRepo    *repository.TaskRepository
	goalRepo    *repository.GoalRepository
	roadmapRepo *repository.RoadmapRepository
	aiService   *AIService
	reviewRepo  *repository.ReviewRepository
}

func NewTaskService(db *pgxpool.Pool, taskRepo *repository.TaskRepository, goalRepo *repository.GoalRepository, roadmapRepo *repository.RoadmapRepository, aiService *AIService, reviewRepo *repository.ReviewRepository) *TaskService {
	return &TaskService{
		db:          db,
		taskRepo:    taskRepo,
		goalRepo:    goalRepo,
		roadmapRepo: roadmapRepo,
		aiService:   aiService,
		reviewRepo:  reviewRepo,
	}
}

func (s *TaskService) StartNewDay(ctx context.Context, userID string) ([]repository.Task, error) {
	log.Println("Memulai proses 'Start New Day' untuk user:", userID)
	now := time.Now().UTC() // Gunakan UTC
today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	yesterday := today.AddDate(0, 0, -1)

	// Pengecekan dependensi untuk debugging
	if s.reviewRepo == nil {
		log.Fatal("FATAL PANIC: TaskService.reviewRepo is nil!")
	}
	if s.taskRepo == nil {
		log.Fatal("FATAL PANIC: TaskService.taskRepo is nil!")
	}

	log.Println("[DEBUG] Langkah 1: Memeriksa review kemarin...")
	_, err := s.reviewRepo.GetReviewByDate(ctx, userID, yesterday)
	log.Println("[DEBUG] Langkah 1 Selesai.")
	
	if err != nil {
		if err == pgx.ErrNoRows {
			log.Println("[DEBUG] Langkah 2: Review kemarin tidak ditemukan, menjalankan 'Lazy Review'...")
			_, _, err_review := s.FinalizeDayReview(ctx, userID, yesterday)
			if err_review != nil {
				log.Printf("[DEBUG] Gagal saat lazy review, tapi tetap lanjut: %v", err_review)
			}
			log.Println("[DEBUG] Langkah 2 Selesai.")
		} else {
			log.Printf("ERROR di Langkah 1: Gagal memeriksa review kemarin: %v", err)
			return nil, err
		}
	}

	log.Println("[DEBUG] Langkah 3: Membuat jadwal untuk hari ini...")
	tasks, err := s.GetOrCreateTodaySchedule(ctx, userID, today)
	if err != nil {
		log.Printf("ERROR di Langkah 3: Gagal membuat jadwal hari ini: %v", err)
		return nil, err
	}
	log.Println("[DEBUG] Langkah 3 Selesai. Proses StartNewDay berhasil.")

	return tasks, nil
}

// GetTodayScheduleReadOnly hanya mengambil tugas untuk tanggal tertentu.
func (s *TaskService) GetTodayScheduleReadOnly(ctx context.Context, userID string, targetDate time.Time) ([]repository.Task, error) {
    return s.taskRepo.GetTasksByDate(ctx, userID, targetDate)
}

func (s *TaskService) GetOrCreateTodaySchedule(ctx context.Context, userID string, targetDate time.Time) ([]repository.Task, error) {
    // Cek tugas yang ada (logika ini tidak berubah)
    existingTasks, err := s.taskRepo.GetTasksByDate(ctx, userID, targetDate)
    if err != nil { return nil, err }
    if len(existingTasks) > 0 { return existingTasks, nil }

    log.Println("[DEBUG] Memulai pembuatan jadwal baru...")

    // Cek Goal Aktif
    activeGoal, err := s.goalRepo.GetActiveGoalByUserID(ctx, userID)
    if err != nil {
        log.Printf("[DEBUG] Error saat mencari goal aktif: %v", err)
        return []repository.Task{}, nil
    }
    if activeGoal == nil {
        log.Println("[DEBUG] Kondisi Gagal: Tidak ada goal aktif ditemukan.")
        return []repository.Task{}, nil
    }
    log.Printf("[DEBUG] Ditemukan Goal Aktif: %s", activeGoal.Description)

    // Cek Langkah Roadmap Berikutnya
    currentStep, err := s.roadmapRepo.GetNextPendingStep(ctx, activeGoal.ID)
    if err != nil {
        if err == pgx.ErrNoRows {
            log.Println("[DEBUG] Kondisi Gagal: Semua langkah roadmap sudah selesai.")
            return []repository.Task{}, nil
        }
        log.Printf("[DEBUG] Error saat mencari langkah roadmap: %v", err)
        return nil, err
    }
    log.Printf("[DEBUG] Ditemukan Langkah Roadmap Aktif: %s", currentStep.Title)

    // Dapatkan tugas kemarin (tidak berubah)
    yesterday := targetDate.AddDate(0, 0, -1)
    yesterdayTasks, err := s.taskRepo.GetTasksByDate(ctx, userID, yesterday)
    if err != nil { return nil, err }

    // Panggil AI
    log.Println("[DEBUG] Memanggil AI Gemini untuk tugas harian...")
    newTasksFromAI, err := s.aiService.GenerateDailyTasksWithAI(ctx, activeGoal.Description, currentStep.Title, yesterdayTasks)
    if err != nil {
        log.Printf("[DEBUG] Error dari panggilan AI: %v", err)
        return nil, err
    }
    log.Printf("[DEBUG] AI berhasil merespons dengan %d tugas.", len(newTasksFromAI))

    if len(newTasksFromAI) == 0 {
        log.Println("[DEBUG] Kondisi Gagal: AI tidak menghasilkan tugas apapun.")
        return []repository.Task{}, nil
    }

    // Simpan tugas ke DB (tidak berubah)
    var createdTasks []repository.Task
    for _, taskToCreate := range newTasksFromAI {
        taskToCreate.UserID = userID
        taskToCreate.Status = "pending"
        taskToCreate.ScheduledDate = targetDate
        taskToCreate.RoadmapStepID = &currentStep.ID

        createdTask, err := s.taskRepo.CreateTask(ctx, &taskToCreate)
        if err != nil { return nil, err }
        createdTasks = append(createdTasks, *createdTask)
    }

    log.Printf("[DEBUG] Berhasil menyimpan %d tugas baru ke DB.", len(createdTasks))
    return createdTasks, nil
}


// FinalizeDayReview sekarang menerima targetDate.
func (s *TaskService) FinalizeDayReview(ctx context.Context, userID string, targetDate time.Time) ([]repository.TaskSummary, string, error) {
    err := s.taskRepo.FinalizeMissedTasks(ctx, userID, targetDate)
	if err != nil { return nil, "", err }

	summary, err := s.taskRepo.GetTaskSummaryByDate(ctx, userID, targetDate)
	if err != nil { return nil, "", err }
	
	activeGoal, _ := s.goalRepo.GetActiveGoalByUserID(ctx, userID)
	if activeGoal == nil { activeGoal = &repository.Goal{ Description: "mencapai tujuan mereka" } }

	feedback, err := s.aiService.GenerateReviewFeedback(ctx, activeGoal.Description, summary)
	if err != nil { feedback = "Tetap semangat untuk esok hari!" }

	review := &repository.DailyReview{
		UserID:      userID,
		ReviewDate:  targetDate,
		Summary:     summary,
		AIFeedback:  feedback,
	}
	if err := s.reviewRepo.CreateOrUpdateReview(ctx, review); err != nil {
		log.Printf("ERROR saving daily review for date %v: %v", targetDate, err)
	}

	return summary, feedback, nil
}

// GetReviewByDate adalah service baru untuk fitur riwayat.
func (s *TaskService) GetReviewByDate(ctx context.Context, userID string, date time.Time) (*repository.DailyReview, error) {
    return s.reviewRepo.GetReviewByDate(ctx, userID, date)
}

// --- FUNGSI-FUNGSI UNTUK MODIFIKASI TUGAS ---
func (s *TaskService) CreateManualTask(ctx context.Context, userID, title string, deadline *time.Time) (*repository.Task, error) {
	now := time.Now().UTC() // Gunakan UTC
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
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