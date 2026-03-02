package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"todo-api/internal/db"
	"todo-api/internal/middleware"
	"todo-api/internal/model"

	"github.com/go-chi/chi/v5"
)

func ListTodos(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	rows, err := db.DB.Query("SELECT id, user_id, title, done, note FROM todos WHERE user_id = ? ORDER BY id DESC", userID)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	todos := make([]model.Todo, 0)
	for rows.Next() {
		var t model.Todo
		var done int
		if err := rows.Scan(&t.ID, &t.UserID, &t.Title, &done, &t.Note); err != nil {
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}
		t.Done = done != 0
		todos = append(todos, t)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(todos)
}

func CreateTodo(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var req model.CreateTodoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Title == "" {
		jsonError(w, "title is required", http.StatusBadRequest)
		return
	}

	result, err := db.DB.Exec("INSERT INTO todos (user_id, title, note) VALUES (?, ?, ?)", userID, req.Title, req.Note)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()
	todo := model.Todo{ID: id, UserID: userID, Title: req.Title, Note: req.Note}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(todo)
}

func UpdateTodo(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	todoID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "invalid todo id", http.StatusBadRequest)
		return
	}

	// Check ownership
	var existing model.Todo
	var done int
	err = db.DB.QueryRow("SELECT id, user_id, title, done, note FROM todos WHERE id = ? AND user_id = ?", todoID, userID).Scan(
		&existing.ID, &existing.UserID, &existing.Title, &done, &existing.Note,
	)
	if err != nil {
		jsonError(w, "todo not found", http.StatusNotFound)
		return
	}
	existing.Done = done != 0

	var req model.UpdateTodoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Title != nil {
		existing.Title = *req.Title
	}
	if req.Done != nil {
		existing.Done = *req.Done
	}
	if req.Note != nil {
		existing.Note = *req.Note
	}

	doneInt := 0
	if existing.Done {
		doneInt = 1
	}

	_, err = db.DB.Exec("UPDATE todos SET title = ?, done = ?, note = ? WHERE id = ? AND user_id = ?",
		existing.Title, doneInt, existing.Note, todoID, userID)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(existing)
}

func DeleteTodo(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	todoID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "invalid todo id", http.StatusBadRequest)
		return
	}

	result, err := db.DB.Exec("DELETE FROM todos WHERE id = ? AND user_id = ?", todoID, userID)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		jsonError(w, "todo not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})
}
