package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/ItsKevinRafaell/go-momentum-api/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

// Definisikan key untuk context
type contextKey string
const UserIDKey contextKey = "user_id"

// JwtMiddleware adalah penjaga gerbang kita
func JwtMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Ambil header Authorization
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		// 2. Periksa format "Bearer <token>"
		headerParts := strings.Split(authHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
			return
		}

		tokenString := headerParts[1]

		// 3. Parse dan validasi token
		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			// Pastikan metode signing adalah HS256
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, http.ErrAbortHandler
			}
			return []byte(config.Get("JWT_SECRET_KEY")), nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// 4. Ambil user_id dari claims dan simpan di context
		userID := claims["sub"].(string)
		ctx := context.WithValue(r.Context(), UserIDKey, userID)

		// 5. Lanjutkan ke handler berikutnya dengan context yang baru
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}