package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"todo-api/internal/middleware"
	"todo-api/internal/model"

	"github.com/go-chi/chi/v5"
)

// TodoHandler holds dependencies for todo endpoints.
type TodoHandler struct {
	DB *sql.DB
}

type createTodoRequest struct {
	Title string `json:"title"`
}

type updateTodoRequest struct {
	Title     *string `json:"title"`
	Completed *bool   `json:"completed"`
}

// List returns all todos for the authenticated user.
func (h *TodoHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	rows, err := h.DB.Query(
		"SELECT id, user_id, title, completed, created_at, updated_at FROM todos WHERE user_id = ? ORDER BY created_at DESC",
		userID,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
		return
	}
	defer rows.Close()

	todos := make([]model.Todo, 0)
	for rows.Next() {
		var t model.Todo
		if err := rows.Scan(&t.ID, &t.UserID, &t.Title, &t.Completed, &t.CreatedAt, &t.UpdatedAt); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
			return
		}
		todos = append(todos, t)
	}
	if err := rows.Err(); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, todos)
}

// Create adds a new todo for the authenticated user.
func (h *TodoHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var req createTodoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "title is required"})
		return
	}

	now := time.Now()
	result, err := h.DB.Exec(
		"INSERT INTO todos (user_id, title, completed, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		userID, req.Title, false, now, now,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
		return
	}

	todo := model.Todo{
		ID:        id,
		UserID:    userID,
		Title:     req.Title,
		Completed: false,
		CreatedAt: now,
		UpdatedAt: now,
	}

	writeJSON(w, http.StatusCreated, todo)
}

// Update modifies an existing todo owned by the authenticated user.
func (h *TodoHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	todoID := chi.URLParam(r, "id")

	// Verify ownership.
	var t model.Todo
	err := h.DB.QueryRow(
		"SELECT id, user_id, title, completed, created_at, updated_at FROM todos WHERE id = ? AND user_id = ?",
		todoID, userID,
	).Scan(&t.ID, &t.UserID, &t.Title, &t.Completed, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "todo not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
		return
	}

	var req updateTodoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "title cannot be empty"})
			return
		}
		t.Title = title
	}
	if req.Completed != nil {
		t.Completed = *req.Completed
	}

	t.UpdatedAt = time.Now()

	_, err = h.DB.Exec(
		"UPDATE todos SET title = ?, completed = ?, updated_at = ? WHERE id = ? AND user_id = ?",
		t.Title, t.Completed, t.UpdatedAt, t.ID, userID,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, t)
}

// Delete removes a todo owned by the authenticated user.
func (h *TodoHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	todoID := chi.URLParam(r, "id")

	result, err := h.DB.Exec("DELETE FROM todos WHERE id = ? AND user_id = ?", todoID, userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
		return
	}

	if rowsAffected == 0 {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "todo not found"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
