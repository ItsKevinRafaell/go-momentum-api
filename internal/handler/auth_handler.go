package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

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

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var payload RegisterPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

    // Menambahkan validasi sederhana untuk input kosong
    if payload.Email == "" || payload.Password == "" {
        http.Error(w, "Email and password cannot be empty", http.StatusBadRequest)
        return
    }

	_, err := h.authService.RegisterUser(r.Context(), payload.Email, payload.Password)
	if err != nil {
		// Di sinilah kita menambahkan logika cerdas kita
		var pgErr *pgconn.PgError
		// errors.As memeriksa apakah `err` adalah tipe PgError
		if errors.As(err, &pgErr) {
			// Kode '23505' adalah kode error standar PostgreSQL untuk "unique_violation"
			if pgErr.Code == "23505" {
				http.Error(w, "Email already exists", http.StatusConflict) // 409 Conflict
				return
			}
		}
		// Jika bukan error duplikat, baru kita anggap sebagai error server
		http.Error(w, "Failed to register user", http.StatusInternalServerError)
		return
	}

	// Jika tidak ada error sama sekali, kirim respons sukses
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "User registered successfully"})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var payload RegisterPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	token, err := h.authService.LoginUser(r.Context(), payload.Email, payload.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// --- PERUBAHAN DIMULAI DI SINI ---
	// Atur cookie di respons
	http.SetCookie(w, &http.Cookie{
		Name:     "token", // Nama cookie kita
		Value:    token,
		Expires:  time.Now().Add(24 * time.Hour), // Samakan dengan masa berlaku token
		HttpOnly: true,     // Paling penting: tidak bisa diakses JavaScript
		Path:     "/",      // Berlaku untuk seluruh situs
		SameSite: http.SameSiteNoneMode, 
		Secure: true,
		// Secure: true,  // Aktifkan ini saat Anda sudah menggunakan HTTPS sepenuhnya
	})

	// Kirim respons sukses tanpa token di body
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Login successful"})
}