package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// Common errors for authentication
var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
)

// HashPassword hashes a password using bcrypt
// Bcrypt is a password hashing function designed to be slow and computationally expensive
// This makes brute-force attacks impractical
func HashPassword(password string) (string, error) {
	// bcrypt.DefaultCost is 10, which means 2^10 iterations
	// This is a good balance between security and performance
	// Higher cost = more secure but slower (12-14 recommended for high security)
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashedBytes), nil
}

// ComparePassword compares a plain text password with a hashed password
// Returns nil if they match, or an error if they don't
// Use this during login to verify the user's password
func ComparePassword(hashedPassword, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		return fmt.Errorf("invalid password: %w", err)
	}
	return nil
}

// Claims represents the JWT token claims
// Claims are the payload of the JWT containing user information
type Claims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

// GenerateToken creates a new JWT token for a user
// JWT (JSON Web Token) is a compact, URL-safe token format
// Structure: header.payload.signature
//   - Header: token type and signing algorithm
//   - Payload: claims (user data)
//   - Signature: cryptographic signature to verify authenticity
func GenerateToken(userID int64, secret string) (string, error) {
	// Set token expiration to 24 hours from now
	// In production, you might want shorter expiration (1-2 hours) with refresh tokens
	expirationTime := time.Now().Add(24 * time.Hour)

	// Create the claims
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			// ExpiresAt: when the token expires
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			// IssuedAt: when the token was created
			IssuedAt: jwt.NewNumericDate(time.Now()),
			// Issuer: who created the token (your application name)
			Issuer: "go-chat",
		},
	}

	// Create token with claims
	// HMAC-SHA256 is used for signing (symmetric key algorithm)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token with the secret key
	// The secret must be kept secure and never exposed to clients
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns the user ID
// This is used by middleware to authenticate requests
func ValidateToken(tokenString, secret string) (int64, error) {
	// Parse the token with claims
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify that the signing method is HMAC
		// This prevents attacks where someone tries to change the algorithm
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		// Return the secret key for validation
		return []byte(secret), nil
	})

	if err != nil {
		return 0, fmt.Errorf("failed to parse token: %w", err)
	}

	// Extract and validate claims
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return 0, ErrInvalidToken
	}

	// Check if token has expired
	// Note: jwt.ParseWithClaims already validates expiration, but we double-check
	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		return 0, ErrExpiredToken
	}

	return claims.UserID, nil
}
