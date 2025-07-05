package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	// Ganti dengan path modul Anda
	"github.com/ItsKevinRafaell/go-momentum-api/internal/auth"
	"github.com/ItsKevinRafaell/go-momentum-api/internal/service"
)

type TaskHandler struct {
	taskService *service.TaskService
}

type UpdateStatusPayload struct {
	Status string `json:"status"`
}

type UpdateDeadlinePayload struct {
	Deadline time.Time `json:"deadline"`
}

type UpdateTitlePayload struct {
	Title string `json:"title"`
}

type CreateTaskPayload struct {
	Title    string     `json:"title"`
	Deadline *time.Time `json:"deadline,omitempty"` // omitempty berarti field ini opsional
}

func NewTaskHandler(taskService *service.TaskService) *TaskHandler {
	return &TaskHandler{taskService: taskService}
}

// StartDay adalah handler untuk endpoint baru yang cerdas.
func (h *TaskHandler) StartDay(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "Invalid token")
		return
	}
	tasks, err := h.taskService.StartNewDay(r.Context(), userID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to start new day")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tasks)
}

// GetTodayScheduleReadOnly sekarang menjadi handler untuk GET.
func (h *TaskHandler) GetTodayScheduleReadOnly(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(auth.UserIDKey).(string)

	now := time.Now().UTC() // Gunakan UTC
today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	tasks, err := h.taskService.GetTodayScheduleReadOnly(r.Context(), userID, today)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to get schedule")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tasks)
}

func (h *TaskHandler) GetTodaySchedule(w http.ResponseWriter, r *http.Request) {
	// 1. Ambil userID dari context
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// 2. Panggil service untuk mendapatkan atau membuat jadwal
	tasks, err := h.taskService.GetOrCreateTodaySchedule(r.Context(), userID, time.Now().UTC(),)
	if err != nil {
		http.Error(w, "Failed to get or create schedule", http.StatusInternalServerError)
		return
	}

	// 3. Kirim response sukses
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tasks)
}

func (h *TaskHandler) UpdateTaskStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}
	taskID := chi.URLParam(r, "taskId")

	var payload UpdateStatusPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := h.taskService.UpdateTaskStatus(r.Context(), userID, taskID, payload.Status)
    if err != nil {
        // --- LOGIKA ERROR BARU ---
        if err == pgx.ErrNoRows {
            http.Error(w, "Task not found or user does not have permission", http.StatusNotFound) // 404 Not Found
            return
        }
        http.Error(w, "Failed to update task status", http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"message": "Task status updated"})
}

func (h *TaskHandler) UpdateTaskDeadline(w http.ResponseWriter, r *http.Request) {
	// Ambil userID dari context
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Ambil taskID dari URL
	taskID := chi.URLParam(r, "taskId")

	// Decode body JSON untuk mendapatkan deadline baru
	var payload UpdateDeadlinePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Panggil service
	err := h.taskService.UpdateTaskDeadline(r.Context(), userID, taskID, payload.Deadline)
	if err != nil {
        // --- LOGIKA ERROR BARU ---
        if err == pgx.ErrNoRows {
            http.Error(w, "Task not found or user does not have permission", http.StatusNotFound) // 404 Not Found
            return
        }
        http.Error(w, "Failed to update task status", http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"message": "Task status updated"})
}


func (h *TaskHandler) UpdateTaskTitle(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}
	taskID := chi.URLParam(r, "taskId")

	var payload UpdateTitlePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := h.taskService.UpdateTaskTitle(r.Context(), userID, taskID, payload.Title)
	if err != nil {
        // --- LOGIKA ERROR BARU ---
        if err == pgx.ErrNoRows {
            http.Error(w, "Task not found or user does not have permission", http.StatusNotFound) // 404 Not Found
            return
        }
        http.Error(w, "Failed to update task status", http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"message": "Task status updated"})
}


func (h *TaskHandler) CreateManualTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	var payload CreateTaskPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
    if payload.Title == "" {
        http.Error(w, "Title cannot be empty", http.StatusBadRequest)
        return
    }

	createdTask, err := h.taskService.CreateManualTask(r.Context(), userID, payload.Title, payload.Deadline)
	if err != nil {
		http.Error(w, "Failed to create task", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // Gunakan status 201 Created untuk resource baru
	json.NewEncoder(w).Encode(createdTask)
}

func (h *TaskHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}
	taskID := chi.URLParam(r, "taskId")

	err := h.taskService.DeleteTask(r.Context(), userID, taskID)
	if err != nil {
		// Jika errornya karena task tidak ditemukan
		if err == pgx.ErrNoRows {
			http.Error(w, "Task not found or user does not have permission", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to delete task", http.StatusInternalServerError)
		return
	}

	// Status 204 No Content adalah respons standar untuk delete yang sukses.
	// Ini memberitahu klien bahwa aksi berhasil dan tidak ada body untuk dikirim kembali.
	w.WriteHeader(http.StatusNoContent)
}

func (h *TaskHandler) ReviewDay(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	summary, feedback, err := h.taskService.FinalizeDayReview(r.Context(), userID, time.Now().UTC())
	if err != nil {
		http.Error(w, "Failed to finalize day review", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"summary":     summary,
		"ai_feedback": feedback,
	})
}

// GetHistoryByDate adalah handler untuk fitur riwayat.
func (h *TaskHandler) GetHistoryByDate(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(auth.UserIDKey).(string)
	dateStr := chi.URLParam(r, "date")

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid date format. Use YYYY-MM-DD.")
		return
	}

	review, err := h.taskService.GetReviewByDate(r.Context(), userID, date)
	if err != nil {
		if err == pgx.ErrNoRows {
			writeJSONError(w, http.StatusNotFound, "No review found for this date.")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "Failed to fetch history.")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(review)
}