package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"todo-api/internal/db"
	"todo-api/internal/handler"
	"todo-api/internal/middleware"

	"github.com/go-chi/chi/v5"
)

func main() {
	database, err := db.InitDB("todo.db")
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}
	defer database.Close()

	authHandler := &handler.AuthHandler{DB: database}
	todoHandler := &handler.TodoHandler{DB: database}

	r := chi.NewRouter()

	// Health check.
	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"}) //nolint:errcheck
	})

	// Public routes.
	r.Post("/api/register", authHandler.Register)
	r.Post("/api/login", authHandler.Login)

	// Protected routes.
	r.Route("/api/todos", func(r chi.Router) {
		r.Use(middleware.JWTAuth)
		r.Get("/", todoHandler.List)
		r.Post("/", todoHandler.Create)
		r.Put("/{id}", todoHandler.Update)
		r.Delete("/{id}", todoHandler.Delete)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("server starting on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
