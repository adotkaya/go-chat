package main

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/drazan344/go-chat/internal/auth"
)

// contextKey is a custom type for context keys to avoid collisions
// Using a custom type prevents conflicts with other packages using context
type contextKey string

const userIDKey contextKey = "userID"

// AuthMiddleware validates JWT tokens and adds user ID to request context
// This middleware protects routes that require authentication
// It expects the token in the Authorization header: "Bearer <token>"
func (app *application) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract the Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeError(w, http.StatusUnauthorized, "missing authorization header")
			return
		}

		// Authorization header format: "Bearer <token>"
		// Split to extract the token part
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			writeError(w, http.StatusUnauthorized, "invalid authorization header format")
			return
		}

		token := parts[1]

		// Validate the token and extract user ID
		userID, err := auth.ValidateToken(token, app.config.auth.jwtSecret)
		if err != nil {
			if errors.Is(err, auth.ErrExpiredToken) {
				writeError(w, http.StatusUnauthorized, "token has expired")
				return
			}
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		// Add user ID to request context
		// Context is Go's way of passing request-scoped values through the call chain
		// The context flows through all handlers and can be accessed anywhere in the request lifecycle
		ctx := context.WithValue(r.Context(), userIDKey, userID)

		// Call the next handler with the updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserIDFromContext extracts the user ID from the request context
// This is used in handlers to get the authenticated user's ID
// Returns an error if the user ID is not found in context (should never happen if middleware is used)
func GetUserIDFromContext(ctx context.Context) (int64, error) {
	userID, ok := ctx.Value(userIDKey).(int64)
	if !ok {
		return 0, errors.New("user ID not found in context")
	}
	return userID, nil
}
