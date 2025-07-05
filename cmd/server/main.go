// file: cmd/server/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

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

	// --- INISIALISASI LENGKAP & BENAR ---

	// 1. Inisialisasi semua Repository
	userRepo := repository.NewUserRepository(dbPool)
	goalRepo := repository.NewGoalRepository(dbPool)
	roadmapRepo := repository.NewRoadmapRepository(dbPool)
	taskRepo := repository.NewTaskRepository(dbPool)
	reviewRepo := repository.NewReviewRepository(dbPool)

	// 2. Inisialisasi semua Service
	aiService := service.NewAIService()
	authService := service.NewAuthService(userRepo)
	goalService := service.NewGoalService(dbPool, goalRepo, roadmapRepo, aiService)
	taskService := service.NewTaskService(dbPool, taskRepo, goalRepo, roadmapRepo, aiService, reviewRepo)

	// 3. Inisialisasi semua Handler
	authHandler := handler.NewAuthHandler(authService)
	goalHandler := handler.NewGoalHandler(goalService)
	taskHandler := handler.NewTaskHandler(taskService)

	// --- AKHIR DARI PERUBAHAN ---

	r := chi.NewRouter()
	r.Use(cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "https://*.vercel.app"}, // Izinkan semua subdomain vercel
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}).Handler)
	r.Use(middleware.Logger)

	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Server is healthy and running!"))
	})

	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
		r.Put("/api/auth/change-password", authHandler.ChangePassword)
	})

	r.Group(func(r chi.Router) {
		r.Use(auth.JwtMiddleware)
		r.Get("/api/auth/me", authHandler.GetCurrentUser)
		r.Post("/api/goals", goalHandler.CreateGoal)
		r.Get("/api/goals/active", goalHandler.GetActiveGoal)
		r.Put("/api/goals/{goalId}", goalHandler.UpdateGoal)
		r.Post("/api/goals/{goalId}/steps", goalHandler.AddRoadmapStep)
		r.Put("/api/roadmap-steps/{stepId}", goalHandler.UpdateRoadmapStep)
		r.Delete("/api/roadmap-steps/{stepId}", goalHandler.DeleteRoadmapStep)
		r.Put("/api/roadmap/reorder", goalHandler.ReorderRoadmapSteps)
		r.Put("/api/roadmap-steps/{stepId}/status", goalHandler.UpdateRoadmapStepStatus)
		
		r.Post("/api/schedule/start-day", taskHandler.StartDay)
		r.Get("/api/schedule/today", taskHandler.GetTodayScheduleReadOnly) // Ganti ke handler read-only
		r.Post("/api/schedule/review", taskHandler.ReviewDay)
		r.Get("/api/schedule/history/{date}", taskHandler.GetHistoryByDate)
		
		r.Post("/api/tasks", taskHandler.CreateManualTask)
		r.Put("/api/tasks/{taskId}", taskHandler.UpdateTaskTitle)
		r.Delete("/api/tasks/{taskId}", taskHandler.DeleteTask)
		r.Put("/api/tasks/{taskId}/status", taskHandler.UpdateTaskStatus)
		r.Put("/api/tasks/{taskId}/deadline", taskHandler.UpdateTaskDeadline)
	})
	
	port := config.Get("API_PORT")
	if port == "" {
		port = "8080"
	}

	listenAddr := fmt.Sprintf("0.0.0.0:%s", port)
	log.Printf("API for Project: Momentum is starting on %s", listenAddr)

	if err := http.ListenAndServe(listenAddr, r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}