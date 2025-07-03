// file: internal/handler/auth_handler.go
package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ItsKevinRafaell/go-momentum-api/internal/auth"
	"github.com/ItsKevinRafaell/go-momentum-api/internal/service"
	"github.com/jackc/pgx/v5/pgconn"
)

type AuthHandler struct {
	authService *service.AuthService
}

type RegisterPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Helper function untuk mengirim error JSON
func writeJSONError(w http.ResponseWriter, statusCode int, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(statusCode)
    json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var payload RegisterPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if payload.Email == "" || payload.Password == "" {
		writeJSONError(w, http.StatusBadRequest, "Email and password cannot be empty")
		return
	}

	_, err := h.authService.RegisterUser(r.Context(), payload.Email, payload.Password)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			writeJSONError(w, http.StatusConflict, "Email already exists")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "Failed to register user")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "User registered successfully"})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
    var payload RegisterPayload
    if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
        writeJSONError(w, http.StatusBadRequest, "Invalid request body")
        return
    }

    token, err := h.authService.LoginUser(r.Context(), payload.Email, payload.Password)
    if err != nil {
        writeJSONError(w, http.StatusUnauthorized, err.Error())
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"token": token})
}

type ChangePasswordPayload struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok {
        writeJSONError(w, http.StatusUnauthorized, "Invalid token")
        return
    }

	var payload ChangePasswordPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	err := h.authService.ChangePassword(r.Context(), userID, payload.OldPassword, payload.NewPassword)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, err.Error())
		return
	}

    w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"message": "Password updated successfully"})
}

func (h *AuthHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
    // Ambil userID dari context yang sudah disisipkan oleh middleware
    userID, ok := r.Context().Value(auth.UserIDKey).(string)
    if !ok {
        writeJSONError(w, http.StatusUnauthorized, "Invalid token")
        return
    }

    // Panggil service untuk mendapatkan detail user berdasarkan ID
    user, err := h.authService.GetUserByID(r.Context(), userID)
    if err != nil {
        writeJSONError(w, http.StatusNotFound, "User not found")
        return
    }

    // Kirim data user sebagai respons JSON
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(user)
}