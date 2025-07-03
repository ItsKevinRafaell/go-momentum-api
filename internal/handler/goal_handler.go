package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/ItsKevinRafaell/go-momentum-api/internal/auth"
	"github.com/ItsKevinRafaell/go-momentum-api/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

type GoalHandler struct {
	goalService *service.GoalService
}

// Payload untuk request pembuatan goal
type CreateGoalPayload struct {
	Description string `json:"description"`
}

type UpdateGoalPayload struct {
    Description string `json:"description"`
}

type AddStepPayload struct {
    Title string `json:"title"`
}

func NewGoalHandler(goalService *service.GoalService) *GoalHandler {
	return &GoalHandler{goalService: goalService}
}

func (h *GoalHandler) CreateGoal(w http.ResponseWriter, r *http.Request) {
	// 1. Ambil userID dari context yang sudah disisipkan oleh middleware JWT
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// 2. Decode body JSON dari request
	var payload CreateGoalPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 3. Panggil service untuk melakukan semua logika
	goal, steps, err := h.goalService.CreateNewGoal(r.Context(), userID, payload.Description)
    if err != nil {
        // --- PERBAIKAN LOGGING DI SINI ---
        // Kita log error aslinya ke terminal server untuk debugging
        log.Printf("ERROR creating goal with AI: %v", err) 
        // Kirim pesan error yang lebih umum ke frontend
        writeJSONError(w, http.StatusInternalServerError, "Failed to generate roadmap from AI.")
        return
    }

	// 4. Kirim response sukses
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"goal":  goal,
		"steps": steps,
	})
}

func (h *GoalHandler) GetActiveGoal(w http.ResponseWriter, r *http.Request) {
	// 1. Ambil userID dari context
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// 2. Panggil service untuk mengambil data
	goal, steps, err := h.goalService.GetActiveGoal(r.Context(), userID)
	if err != nil {
		http.Error(w, "Failed to get active goal", http.StatusInternalServerError)
		return
	}

	// 3. Jika tidak ada goal yang ditemukan, kirim response kosong atau pesan
	if goal == nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("No active goal found"))
		return
	}

	// 4. Kirim response sukses dengan data yang ditemukan
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"goal":  goal,
		"steps": steps,
	})
}

func (h *GoalHandler) UpdateGoal(w http.ResponseWriter, r *http.Request) {
    userID, ok := r.Context().Value(auth.UserIDKey).(string)
    if !ok { writeJSONError(w, http.StatusUnauthorized, "Invalid token"); return }

    goalID := chi.URLParam(r, "goalId") // Ambil ID dari URL

    var payload UpdateGoalPayload
    if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
        writeJSONError(w, http.StatusBadRequest, "Invalid request body")
        return
    }

    goal, steps, err := h.goalService.UpdateGoal(r.Context(), userID, goalID, payload.Description)
    if err != nil {
        writeJSONError(w, http.StatusInternalServerError, "Failed to update goal")
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "goal":  goal,
        "steps": steps,
    })
}

func (h *GoalHandler) AddRoadmapStep(w http.ResponseWriter, r *http.Request) {
    goalID := chi.URLParam(r, "goalId")

    var payload AddStepPayload
    if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
        writeJSONError(w, http.StatusBadRequest, "Invalid request body")
        return
    }
    if payload.Title == "" {
        writeJSONError(w, http.StatusBadRequest, "Title cannot be empty")
        return
    }

    newStep, err := h.goalService.AddRoadmapStep(r.Context(), goalID, payload.Title)
    if err != nil {
        writeJSONError(w, http.StatusInternalServerError, "Failed to add roadmap step")
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(newStep)
}

func (h *GoalHandler) UpdateRoadmapStep(w http.ResponseWriter, r *http.Request) {
    userID, ok := r.Context().Value(auth.UserIDKey).(string)
    if !ok {
        writeJSONError(w, http.StatusUnauthorized, "Invalid token")
        return
    }
    stepID := chi.URLParam(r, "stepId")

    var payload AddStepPayload // Bisa pakai payload yang sama (hanya butuh title)
    if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
        writeJSONError(w, http.StatusBadRequest, "Invalid request body")
        return
    }
    if payload.Title == "" {
        writeJSONError(w, http.StatusBadRequest, "Title cannot be empty")
        return
    }

    err := h.goalService.UpdateRoadmapStep(r.Context(), userID, stepID, payload.Title)
    if err != nil {
        if err == pgx.ErrNoRows {
            writeJSONError(w, http.StatusNotFound, "Roadmap step not found or you don't have permission")
            return
        }
        writeJSONError(w, http.StatusInternalServerError, "Failed to update roadmap step")
        return
    }

    writeJSONError(w, http.StatusOK, "Roadmap step updated successfully")
}