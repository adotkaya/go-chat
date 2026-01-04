package main

import (
	"database/sql"
	"errors"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	ws "github.com/drazan344/go-chat/internal/websocket"
)

// upgrader configures the WebSocket upgrade
var upgrader = websocket.Upgrader{
	// ReadBufferSize and WriteBufferSize specify I/O buffer sizes
	// 1024 bytes is sufficient for our chat messages
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,

	// CheckOrigin returns true to allow connections from any origin
	// In production, you should validate the origin to prevent CSRF attacks
	// Example: return r.Header.Get("Origin") == "https://yourdomain.com"
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins (for development)
	},
}

// websocketHandler handles WebSocket upgrade and connection
// GET /v1/rooms/{roomID}/ws
// Requires authentication (JWT token)
// The user must be a member of the room to connect
func (app *application) websocketHandler(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user ID from context
	userID, err := GetUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Extract room ID from URL
	roomID, err := extractIDFromURL(r, "roomID")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Verify user is a member of the room
	// Users can only connect to rooms they've joined
	isMember, err := app.store.RoomMembers.IsUserInRoom(r.Context(), roomID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to verify room membership")
		return
	}
	if !isMember {
		writeError(w, http.StatusForbidden, "you must join the room before connecting")
		return
	}

	// Get user information to include username in messages
	user, err := app.store.Users.GetByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to retrieve user")
		return
	}

	// Upgrade HTTP connection to WebSocket
	// This switches the protocol from HTTP to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Create a new client for this connection
	client := &ws.Client{
		hub:      app.hub,
		conn:     conn,
		send:     make(chan []byte, 256), // Buffered channel to prevent blocking
		userID:   userID,
		username: user.Username,
		roomID:   roomID,
	}

	// Register the client with the hub
	// This adds the client to the room's client list
	app.hub.register <- client

	// Start goroutines for reading and writing
	// These run concurrently to handle bidirectional communication
	// readPump: reads messages from WebSocket and sends to hub
	// writePump: reads from send channel and writes to WebSocket
	go client.writePump()
	go client.readPump()

	log.Printf("WebSocket connection established: user=%s room=%d", user.Username, roomID)
}
