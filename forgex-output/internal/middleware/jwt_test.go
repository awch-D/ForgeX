package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGenerateAndValidateToken(t *testing.T) {
	token, err := GenerateToken(42)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
}

func TestJWTAuth_ValidToken(t *testing.T) {
	token, err := GenerateToken(99)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	var capturedUID int64
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUID = GetUserID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	JWTAuth(inner).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if capturedUID != 99 {
		t.Fatalf("expected user_id 99, got %d", capturedUID)
	}
}

func TestJWTAuth_MissingHeader(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	JWTAuth(inner).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}

	var body map[string]string
	json.NewDecoder(rr.Body).Decode(&body)
	if body["error"] == "" {
		t.Fatal("expected error message in response")
	}
}

func TestJWTAuth_InvalidToken(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	rr := httptest.NewRecorder()

	JWTAuth(inner).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestJWTAuth_BadFormat(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic abc123")
	rr := httptest.NewRecorder()

	JWTAuth(inner).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestGetUserID_NoValue(t *testing.T) {
	ctx := context.Background()
	if uid := GetUserID(ctx); uid != 0 {
		t.Fatalf("expected 0, got %d", uid)
	}
}
