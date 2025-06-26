package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/ItsKevinRafaell/go-momentum-api/internal/auth"
	"github.com/ItsKevinRafaell/go-momentum-api/internal/config"
	"github.com/ItsKevinRafaell/go-momentum-api/internal/database"
	"github.com/ItsKevinRafaell/go-momentum-api/internal/handler"
	"github.com/ItsKevinRafaell/go-momentum-api/internal/repository"
	"github.com/ItsKevinRafaell/go-momentum-api/internal/service"
)

func main() {
    config.LoadConfig()
    dbPool := database.NewConnection(context.Background())
    defer dbPool.Close()

    log.Println("Database connection established successfully")

    userRepo := repository.NewUserRepository(dbPool)
    authService := service.NewAuthService(userRepo)
    authHandler := handler.NewAuthHandler(authService)

    r := chi.NewRouter()
    r.Use(middleware.Logger)

    r.Route("/api/auth", func(r chi.Router) {
        r.Post("/register", authHandler.Register)
        r.Post("/login", authHandler.Login)
    })

    r.Group(func(r chi.Router) {
        r.Use(auth.JwtMiddleware)

        r.Post("/api/goals", func(w http.ResponseWriter, r *http.Request) {
			// Ambil userID dari context yang sudah disisipkan oleh middleware
			userID := r.Context().Value(auth.UserIDKey).(string)
			w.Write([]byte("Welcome! Your User ID is: " + userID))
		})
    })

    port := config.Get("API_PORT")
    if port == "" {
        port = "8080" // Default port if not set
    }
    fmt.Println("API for Project: Momentum is starting...")

    if err:= http.ListenAndServe(":"+port, r); err != nil {
        log.Fatalf("Failed to start server: %v", err)
    }
}