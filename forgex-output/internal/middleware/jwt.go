package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// contextKey is an unexported type for context keys to avoid collisions.
type contextKey struct{}

var userIDKey = contextKey{}

const defaultJWTSecret = "default-secret-change-me"

func getJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = defaultJWTSecret
	}
	return []byte(secret)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// JWTAuth is a chi-compatible middleware that validates a Bearer JWT token
// from the Authorization header and stores the user_id claim in the request context.
func JWTAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeJSONError(w, http.StatusUnauthorized, "missing authorization header")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeJSONError(w, http.StatusUnauthorized, "invalid authorization header format")
			return
		}

		tokenString := parts[1]

		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return getJWTSecret(), nil
		})
		if err != nil || !token.Valid {
			writeJSONError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "invalid token claims")
			return
		}

		userIDFloat, ok := claims["user_id"].(float64)
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "missing user_id claim")
			return
		}

		userID := int64(userIDFloat)
		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GenerateToken creates a signed JWT for the given user ID with a 24-hour expiry.
func GenerateToken(userID int64) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getJWTSecret())
}

// GetUserID extracts the user ID from the request context.
// Returns 0 if no user ID is present.
func GetUserID(ctx context.Context) int64 {
	uid, ok := ctx.Value(userIDKey).(int64)
	if !ok {
		return 0
	}
	return uid
}
