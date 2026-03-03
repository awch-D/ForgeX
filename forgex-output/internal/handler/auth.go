package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"todo-api/internal/middleware"

	"golang.org/x/crypto/bcrypt"
)

// AuthHandler holds dependencies for authentication endpoints.
type AuthHandler struct {
	DB *sql.DB
}

type registerRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type registerResponse struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

// Register handles user registration.
// Accepts JSON {username, password}, hashes the password with bcrypt,
// inserts into the users table, and returns 201 with {id, username}.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Password = strings.TrimSpace(req.Password)

	if req.Username == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "username and password are required"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
		return
	}

	result, err := h.DB.Exec(
		"INSERT INTO users (username, password_hash) VALUES (?, ?)",
		req.Username, string(hashedPassword),
	)
	if err != nil {
		if isDuplicateKeyError(err) {
			writeJSON(w, http.StatusConflict, errorResponse{Error: "username already exists"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
		return
	}

	writeJSON(w, http.StatusCreated, registerResponse{ID: id, Username: req.Username})
}

// Login handles user authentication.
// Accepts JSON {username, password}, looks up the user by username,
// compares the bcrypt hash, generates a JWT via middleware.GenerateToken,
// and returns 200 with {token}.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Password = strings.TrimSpace(req.Password)

	if req.Username == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "username and password are required"})
		return
	}

	var id int64
	var hashedPassword string
	err := h.DB.QueryRow(
		"SELECT id, password_hash FROM users WHERE username = ?",
		req.Username,
	).Scan(&id, &hashedPassword)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "invalid credentials"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(req.Password)); err != nil {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "invalid credentials"})
		return
	}

	token, err := middleware.GenerateToken(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, loginResponse{Token: token})
}

// isDuplicateKeyError checks whether the error indicates a unique constraint violation.
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "23505")
}
