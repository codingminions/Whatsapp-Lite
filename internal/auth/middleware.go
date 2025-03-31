package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"chat-app/internal/models"
	"chat-app/pkg/logger"
	"chat-app/pkg/token"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

// UserIDKey is the key for user ID in context
const UserIDKey contextKey = "user_id"

// UsernameKey is the key for username in context
const UsernameKey contextKey = "username"

// AuthMiddleware struct holds dependencies for the auth middleware
type AuthMiddleware struct {
	tokenMaker token.Maker
	logger     logger.Logger
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(tokenMaker token.Maker, logger logger.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		tokenMaker: tokenMaker,
		logger:     logger,
	}
}

// Authenticate middleware for HTTP handlers
func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			sendJSON(w, http.StatusUnauthorized, models.ErrorResponse{
				Code:    1008,
				Message: "Authentication required",
			})
			m.logger.Info("Authentication failed: no token provided")
			return
		}

		// Check if the header starts with "Bearer "
		fields := strings.Fields(authHeader)
		if len(fields) != 2 || fields[0] != "Bearer" {
			sendJSON(w, http.StatusUnauthorized, models.ErrorResponse{
				Code:    1008,
				Message: "Invalid authorization header format",
			})
			m.logger.Info("Authentication failed: invalid header format")
			return
		}

		// Verify token
		payload, err := m.tokenMaker.VerifyToken(fields[1])
		if err != nil {
			var vErr token.ValidationError
			if errors.As(err, &vErr) {
				sendJSON(w, http.StatusUnauthorized, models.ErrorResponse{
					Code:    1008,
					Message: vErr.Error(),
				})
			} else {
				sendJSON(w, http.StatusUnauthorized, models.ErrorResponse{
					Code:    1008,
					Message: "Invalid token",
				})
			}
			m.logger.Info("Authentication failed: invalid token", "error", err)
			return
		}

		// Add user info to context
		ctx := context.WithValue(r.Context(), UserIDKey, payload.UserID)
		ctx = context.WithValue(ctx, UsernameKey, payload.Username)

		// Call the next handler with the updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserID extracts the user ID from the request context
func GetUserID(ctx context.Context) (string, error) {
	userID, ok := ctx.Value(UserIDKey).(string)
	if !ok {
		return "", errors.New("user ID not found in context")
	}
	return userID, nil
}

// GetUsername extracts the username from the request context
func GetUsername(ctx context.Context) (string, error) {
	username, ok := ctx.Value(UsernameKey).(string)
	if !ok {
		return "", errors.New("username not found in context")
	}
	return username, nil
}

// sendJSON sends a JSON response
func sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			fmt.Printf("Error encoding JSON: %v\n", err)
		}
	}
}
