// file: cmd/server/main.go

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors" // Pastikan Anda mengimpor modul CORS

	// Ganti dengan path modul Anda
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
	goalRepo := repository.NewGoalRepository(dbPool)
	roadmapRepo := repository.NewRoadmapRepository(dbPool)
	taskRepo := repository.NewTaskRepository(dbPool)

	// 2. Buat instance AIService
	aiService := service.NewAIService()

	// 3. Berikan aiService ke service lain yang membutuhkannya
	authService := service.NewAuthService(userRepo)
	goalService := service.NewGoalService(goalRepo, roadmapRepo, aiService)
	taskService := service.NewTaskService(taskRepo, goalRepo, roadmapRepo, aiService)

	// 4. Buat instance Handler seperti biasa
	authHandler := handler.NewAuthHandler(authService)
	goalHandler := handler.NewGoalHandler(goalService)
	taskHandler := handler.NewTaskHandler(taskService)


	r := chi.NewRouter()

	// --- TAMBAHKAN BLOK CORS DI SINI ---
	r.Use(cors.New(cors.Options{
		// Sesuaikan daftar origin ini dengan kebutuhan development Anda
		AllowedOrigins:   []string{"http://localhost:3000", "http://192.168.248.1:3000", "https://momentum-next-js.vercel.app"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}).Handler)
	// --- AKHIR BLOK CORS ---

	r.Use(middleware.Logger)

	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Server is healthy and running!"))
	})

	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
	})

	r.Group(func(r chi.Router) {
		r.Use(auth.JwtMiddleware)

		r.Put("/api/auth/change-password", authHandler.ChangePassword)

		r.Post("/api/goals", goalHandler.CreateGoal)
		r.Get("/api/goals/active", goalHandler.GetActiveGoal)
		r.Put("/api/goals/{goalId}", goalHandler.UpdateGoal)

		r.Get("/api/schedule/today", taskHandler.GetTodaySchedule)
		r.Put("/api/tasks/{taskId}/status", taskHandler.UpdateTaskStatus)
		r.Put("/api/tasks/{taskId}/deadline", taskHandler.UpdateTaskDeadline)
		r.Put("/api/tasks/{taskId}", taskHandler.UpdateTaskTitle)
		r.Post("/api/tasks", taskHandler.CreateManualTask)
		r.Delete("/api/tasks/{taskId}", taskHandler.DeleteTask)
		r.Post("/api/schedule/review", taskHandler.ReviewDay)
	})

	port := config.Get("API_PORT")
	if port == "" {
		port = "8080" // Default port if not set
	}

	listenAddr := fmt.Sprintf("0.0.0.0:%s", port)
	log.Printf("API for Project: Momentum is starting on %s", listenAddr)

	if err := http.ListenAndServe(listenAddr, r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}