package main

import (
	"log"
	"net/http"
	"os"

	"todo-api/internal/db"
	"todo-api/internal/handler"
	"todo-api/internal/middleware"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

func main() {
	// JWT secret from env or default
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "change-me-in-production"
	}
	middleware.JWTSecret = []byte(secret)

	// Database
	dsn := os.Getenv("DB_PATH")
	if dsn == "" {
		dsn = "todo.db"
	}
	db.Init(dsn)

	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)

	// Public routes
	r.Post("/api/register", handler.Register)
	r.Post("/api/login", handler.Login)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth)
		r.Get("/api/todos", handler.ListTodos)
		r.Post("/api/todos", handler.CreateTodo)
		r.Put("/api/todos/{id}", handler.UpdateTodo)
		r.Delete("/api/todos/{id}", handler.DeleteTodo)
	})

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}
	log.Printf("server starting on %s", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}
