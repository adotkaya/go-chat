package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// writeJSON writes a JSON response to the client
// This helper standardizes JSON responses across all handlers
// The status parameter sets the HTTP status code (200, 201, 400, etc.)
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	// Set Content-Type header to indicate JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	// Encode data as JSON and write to response
	// If encoding fails, there's not much we can do since headers are already sent
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Log the error but can't send error to client anymore
		fmt.Printf("Error encoding JSON: %v\n", err)
	}
}

// readJSON reads and unmarshals JSON from the request body
// The dst parameter should be a pointer to the struct you want to unmarshal into
// Example: var req LoginRequest; readJSON(r, &req)
func readJSON(r *http.Request, dst interface{}) error {
	// Limit request body size to prevent DOS attacks
	// 1MB should be plenty for our JSON payloads
	maxBytes := 1_048_576 // 1MB
	r.Body = http.MaxBytesReader(nil, r.Body, int64(maxBytes))

	// Create JSON decoder
	decoder := json.NewDecoder(r.Body)

	// DisallowUnknownFields makes the decoder return an error if the JSON contains
	// fields that don't match the destination struct
	// This helps catch typos and prevents clients from sending unexpected data
	decoder.DisallowUnknownFields()

	// Decode JSON into the destination
	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	return nil
}

// writeError writes a standardized error response
// This ensures all error responses have the same format
func writeError(w http.ResponseWriter, status int, message string) {
	// Create a simple error response structure
	type errorResponse struct {
		Error string `json:"error"`
	}

	writeJSON(w, status, errorResponse{Error: message})
}

// extractIDFromURL extracts an integer ID from URL parameters
// This is commonly used for routes like /rooms/{roomID} where roomID needs to be parsed
// The param parameter is the URL parameter name (e.g., "roomID")
func extractIDFromURL(r *http.Request, param string) (int64, error) {
	// Chi stores URL parameters in the request context
	idStr := chi.URLParam(r, param)
	if idStr == "" {
		return 0, fmt.Errorf("missing %s parameter", param)
	}

	// Parse string to int64
	// ParseInt parameters: string, base (10 for decimal), bitSize (64 for int64)
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s parameter: must be an integer", param)
	}

	return id, nil
}

// HTTP Status Codes Reference (for educational purposes):
//
// 2xx Success:
//   200 OK - Request succeeded
//   201 Created - Resource created successfully
//   204 No Content - Success but no content to return
//
// 4xx Client Errors:
//   400 Bad Request - Invalid request (validation failed)
//   401 Unauthorized - Authentication required or failed
//   403 Forbidden - Authenticated but not authorized
//   404 Not Found - Resource doesn't exist
//   409 Conflict - Request conflicts with current state (e.g., duplicate email)
//
// 5xx Server Errors:
//   500 Internal Server Error - Unexpected server error
//   503 Service Unavailable - Server temporarily unavailable
