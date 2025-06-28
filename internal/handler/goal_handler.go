package handler

import (
	"encoding/json"
	"net/http"

	"github.com/ItsKevinRafaell/go-momentum-api/internal/auth"
	"github.com/ItsKevinRafaell/go-momentum-api/internal/service"
)

type GoalHandler struct {
	goalService *service.GoalService
}

// Payload untuk request pembuatan goal
type CreateGoalPayload struct {
	Description string `json:"description"`
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
		// Nanti kita bisa buat error handling yang lebih spesifik
		http.Error(w, "Failed to create goal", http.StatusInternalServerError)
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