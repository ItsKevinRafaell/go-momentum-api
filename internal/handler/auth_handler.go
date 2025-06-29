package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/ItsKevinRafaell/go-momentum-api/internal/auth"
	"github.com/ItsKevinRafaell/go-momentum-api/internal/config"
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
        writeJSONError(w, http.StatusBadRequest, "Invalid request body") // Gunakan helper
        return
    }
    if payload.Email == "" || payload.Password == "" {
        writeJSONError(w, http.StatusBadRequest, "Email and password cannot be empty") // Gunakan helper
        return
    }

    _, err := h.authService.RegisterUser(r.Context(), payload.Email, payload.Password)
    if err != nil {
        var pgErr *pgconn.PgError
        if errors.As(err, &pgErr) && pgErr.Code == "23505" {
            writeJSONError(w, http.StatusConflict, "Email already exists") // Gunakan helper
            return
        }
        writeJSONError(w, http.StatusInternalServerError, "Failed to register user") // Gunakan helper
        return
    }

    // ... (kode response sukses tetap sama) ...
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
        // --- GUNAKAN HELPER BARU KITA ---
        writeJSONError(w, http.StatusUnauthorized, err.Error())
        return
    }

	// --- INI BAGIAN UTAMA PERUBAHANNYA ---
	// Membuat cookie baru untuk token
	cookie := &http.Cookie{
		Name:     "token",
		Value:    token,
		Expires:  time.Now().Add(24 * time.Hour), // Cookie berlaku 24 jam
		HttpOnly: true,     // Kunci keamanan: tidak bisa diakses JavaScript
		Path:     "/",      // Berlaku untuk seluruh situs
		SameSite: http.SameSiteLaxMode, // Mode Lax untuk keamanan
		Secure:   false,    // Non-secure untuk development, ubah ke true
	}
	
	// Membuat cookie "pintar" berdasarkan lingkungan (environment)
	// if config.Get("APP_ENV") == "production" {
	// 	// Untuk produksi (di Fly.io), cookie harus Secure dan SameSite=None
	// 	cookie.SameSite = http.SameSiteNoneMode
	// 	cookie.Secure = true
	// } else {
	// 	// Untuk development (di lokal), gunakan LaxMode dan non-secure
	// 	cookie.SameSite = http.SameSiteLaxMode
	// 	cookie.Secure = false
	// }

	// Terapkan cookie ke dalam response header
	http.SetCookie(w, cookie)
	// --- AKHIR DARI PERUBAHAN ---

	// Kirim response sukses tanpa token di body
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Login successful"})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Buat cookie dengan waktu kedaluwarsa di masa lalu untuk menghapusnya
	cookie := &http.Cookie{
		Name:     "token",
		Value:    "", // Kosongkan nilainya
		Expires:  time.Now().Add(-time.Hour), // Atur ke satu jam yang lalu
		HttpOnly: true,
		Path:     "/",
	}

	// Gunakan pengaturan yang sama dengan saat login untuk konsistensi
	if config.Get("APP_ENV") == "production" {
		cookie.SameSite = http.SameSiteNoneMode
		cookie.Secure = true
	} else {
		cookie.SameSite = http.SameSiteLaxMode
		cookie.Secure = false
	}

	http.SetCookie(w, cookie)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Successfully logged out"})
}

func writeJSONError(w http.ResponseWriter, statusCode int, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(statusCode)
    json.NewEncoder(w).Encode(map[string]string{"error": message})
}

type ChangePasswordPayload struct {
    OldPassword string `json:"old_password"`
    NewPassword string `json:"new_password"`
}

func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
    userID, ok := r.Context().Value(auth.UserIDKey).(string)
    if !ok { return /* handle error */ }

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

    writeJSONError(w, http.StatusOK, "Password updated successfully")
}