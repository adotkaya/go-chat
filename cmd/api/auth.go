package main

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/drazan344/go-chat/internal/auth"
	"github.com/drazan344/go-chat/internal/store"
)

// RegisterRequest represents the JSON structure for user registration
type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest represents the JSON structure for user login
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse represents the response after successful login/registration
// It includes the JWT token and user information
type AuthResponse struct {
	Token string      `json:"token"`
	User  *store.User `json:"user"`
}

// registerHandler handles user registration
// POST /v1/auth/register
// Request body: {"username": "john", "email": "john@example.com", "password": "secret123"}
// Response: {"token": "jwt...", "user": {...}}
func (app *application) registerHandler(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var req RegisterRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate input
	// Basic validation - in production, you might want more thorough validation
	if req.Username == "" || req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "username, email, and password are required")
		return
	}

	// Validate email format (basic check)
	if !strings.Contains(req.Email, "@") {
		writeError(w, http.StatusBadRequest, "invalid email format")
		return
	}

	// Validate password strength (at least 6 characters for this demo)
	// In production, enforce stronger password requirements
	if len(req.Password) < 6 {
		writeError(w, http.Status BadRequest, "password must be at least 6 characters")
		return
	}

	// Hash the password before storing
	// NEVER store plain text passwords!
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to process password")
		return
	}

	// Create user in database
	user := &store.User{
		Username: req.Username,
		Email:    req.Email,
		Password: hashedPassword, // Store hashed password, not plain text
	}

	// Use context from request for database operations
	// This allows for timeout and cancellation
	if err := app.store.Users.Create(r.Context(), user); err != nil {
		// Check if error is due to unique constraint violation (duplicate email/username)
		// Different databases return different errors, but the message usually contains "unique" or "duplicate"
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			writeError(w, http.StatusConflict, "email or username already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	// Generate JWT token for the new user
	token, err := auth.GenerateToken(user.ID, app.config.auth.jwtSecret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	// Clear password before sending response
	// Even though it's hashed, we don't want to send it to the client
	user.Password = ""

	// Return success response with token and user info
	// 201 Created is the appropriate status code for resource creation
	writeJSON(w, http.StatusCreated, AuthResponse{
		Token: token,
		User:  user,
	})
}

// loginHandler handles user authentication
// POST /v1/auth/login
// Request body: {"email": "john@example.com", "password": "secret123"}
// Response: {"token": "jwt...", "user": {...}}
func (app *application) loginHandler(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var req LoginRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate input
	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	// Find user by email
	user, err := app.store.Users.GetByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Don't reveal whether email exists or not for security
			// Use generic error message
			writeError(w, http.StatusUnauthorized, "invalid email or password")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to retrieve user")
		return
	}

	// Compare provided password with hashed password in database
	// This uses bcrypt's built-in comparison which handles the salt automatically
	if err := auth.ComparePassword(user.Password, req.Password); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	// Generate JWT token for the authenticated user
	token, err := auth.GenerateToken(user.ID, app.config.auth.jwtSecret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	// Clear password before sending response
	user.Password = ""

	// Return success response with token and user info
	// 200 OK is appropriate for successful login
	writeJSON(w, http.StatusOK, AuthResponse{
		Token: token,
		User:  user,
	})
}

// getCurrentUserHandler returns the currently authenticated user's information
// GET /v1/auth/me
// Requires authentication (JWT token in Authorization header)
// Response: {"id": 1, "username": "john", "email": "john@example.com", ...}
func (app *application) getCurrentUserHandler(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context (set by AuthMiddleware)
	userID, err := GetUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Retrieve user from database
	user, err := app.store.Users.GetByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to retrieve user")
		return
	}

	// Clear password before sending response
	user.Password = ""

	// Return user information
	writeJSON(w, http.StatusOK, user)
}
