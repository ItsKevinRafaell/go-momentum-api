// file: internal/auth/middleware.go
package auth

import (
	"context"
	"net/http"

	"github.com/ItsKevinRafaell/go-momentum-api/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

// Definisikan key untuk context
type contextKey string
const UserIDKey contextKey = "user_id"

// JwtMiddleware adalah penjaga gerbang kita yang sudah diperbaiki
func JwtMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		
		// --- LOGIKA BARU UNTUK MEMBACA COOKIE ---
		tokenCookie, err := r.Cookie("token")
		if err != nil {
			// Jika cookie tidak ada sama sekali
			if err == http.ErrNoCookie {
				http.Error(w, "Authorization cookie required", http.StatusUnauthorized)
				return
			}
			// Error lain
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		// Ambil nilai token dari cookie
		tokenString := tokenCookie.Value
		// --- AKHIR LOGIKA BARU ---

		// Sisa logika untuk memvalidasi token tetap sama
		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, http.ErrAbortHandler
			}
			return []byte(config.Get("JWT_SECRET_KEY")), nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		userID, ok := claims["sub"].(string)
		if !ok {
			http.Error(w, "Invalid token claims", http.StatusUnauthorized)
			return
		}
		
		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}