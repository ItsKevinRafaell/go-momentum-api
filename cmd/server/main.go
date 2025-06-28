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

    goalRepo := repository.NewGoalRepository(dbPool)
    roadmapRepo := repository.NewRoadmapRepository(dbPool)
    goalService := service.NewGoalService(goalRepo, roadmapRepo)
    goalHandler := handler.NewGoalHandler(goalService)

    taskRepo := repository.NewTaskRepository(dbPool)
    taskService := service.NewTaskService(taskRepo, goalRepo, roadmapRepo)
    taskHandler := handler.NewTaskHandler(taskService)

    r := chi.NewRouter()
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

        r.Post("/api/goals", goalHandler.CreateGoal)
        r.Get("/api/goals/active", goalHandler.GetActiveGoal)

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
    fmt.Println("API for Project: Momentum is starting...")


    listenAddr := fmt.Sprintf("0.0.0.0:%s", port)
    log.Printf("API for Project: Momentum is starting on %s", listenAddr)

    if err := http.ListenAndServe(listenAddr, r); err != nil {
        log.Fatalf("Failed to start server: %v", err)
    }
}

