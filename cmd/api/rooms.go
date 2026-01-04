package main

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/drazan344/go-chat/internal/store"
)

// CreateRoomRequest represents the JSON structure for creating a room
type CreateRoomRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// createRoomHandler creates a new chat room
// POST /v1/rooms
// Requires authentication
// Request body: {"name": "general", "description": "General chat room"}
// Response: {"id": 1, "name": "general", ...}
func (app *application) createRoomHandler(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user ID from context
	userID, err := GetUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Parse request body
	var req CreateRoomRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate input
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "room name is required")
		return
	}

	// Room names should be lowercase and URL-friendly (like Slack channels)
	// Convert to lowercase and trim spaces
	req.Name = strings.ToLower(strings.TrimSpace(req.Name))

	// Create room in database
	room := &store.Room{
		Name:        req.Name,
		Description: req.Description,
		CreatedBy:   userID,
	}

	if err := app.store.Rooms.Create(r.Context(), room); err != nil {
		// Check for duplicate room name
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			writeError(w, http.StatusConflict, "room name already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create room")
		return
	}

	// Automatically join the creator to the room
	// This makes sense as the creator would want to be in their own room
	if err := app.store.RoomMembers.Join(r.Context(), room.ID, userID); err != nil {
		// Room was created but join failed - log this but don't fail the request
		// The user can manually join later
		writeError(w, http.StatusInternalServerError, "room created but failed to join")
		return
	}

	// Return the created room with 201 Created status
	writeJSON(w, http.StatusCreated, room)
}

// listRoomsHandler returns all available chat rooms
// GET /v1/rooms
// Requires authentication
// Response: [{"id": 1, "name": "general", ...}, {"id": 2, "name": "random", ...}]
func (app *application) listRoomsHandler(w http.ResponseWriter, r *http.Request) {
	// Get all rooms from database
	// In a production app with many rooms, you'd want pagination here
	rooms, err := app.store.Rooms.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to retrieve rooms")
		return
	}

	// Return empty array instead of null if no rooms exist
	// This is better for client-side code
	if rooms == nil {
		rooms = []*store.Room{}
	}

	writeJSON(w, http.StatusOK, rooms)
}

// getRoomHandler returns details about a specific room
// GET /v1/rooms/{roomID}
// Requires authentication
// Response: {"id": 1, "name": "general", ...}
func (app *application) getRoomHandler(w http.ResponseWriter, r *http.Request) {
	// Extract room ID from URL
	roomID, err := extractIDFromURL(r, "roomID")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Get room from database
	room, err := app.store.Rooms.GetByID(r.Context(), roomID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "room not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to retrieve room")
		return
	}

	writeJSON(w, http.StatusOK, room)
}

// joinRoomHandler adds the current user to a room
// POST /v1/rooms/{roomID}/join
// Requires authentication
// Response: {"message": "joined room successfully"}
func (app *application) joinRoomHandler(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user ID
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

	// Verify room exists
	_, err = app.store.Rooms.GetByID(r.Context(), roomID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "room not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to verify room")
		return
	}

	// Join the room
	if err := app.store.RoomMembers.Join(r.Context(), roomID, userID); err != nil {
		// Check if already a member (duplicate key error)
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			writeError(w, http.StatusConflict, "already a member of this room")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to join room")
		return
	}

	// Return success message
	type response struct {
		Message string `json:"message"`
	}
	writeJSON(w, http.StatusOK, response{Message: "joined room successfully"})
}

// leaveRoomHandler removes the current user from a room
// POST /v1/rooms/{roomID}/leave
// Requires authentication
// Response: {"message": "left room successfully"}
func (app *application) leaveRoomHandler(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user ID
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

	// Leave the room
	// This is idempotent - if user is not a member, it silently succeeds
	if err := app.store.RoomMembers.Leave(r.Context(), roomID, userID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to leave room")
		return
	}

	// Return success message
	type response struct {
		Message string `json:"message"`
	}
	writeJSON(w, http.StatusOK, response{Message: "left room successfully"})
}

// getRoomMessagesHandler retrieves message history for a room
// GET /v1/rooms/{roomID}/messages
// Requires authentication and room membership
// Response: [{"id": 1, "content": "Hello!", "username": "john", ...}, ...]
func (app *application) getRoomMessagesHandler(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user ID
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

	// Check if user is a member of the room
	// Users can only see messages in rooms they've joined
	isMember, err := app.store.RoomMembers.IsUserInRoom(r.Context(), roomID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to verify room membership")
		return
	}
	if !isMember {
		writeError(w, http.StatusForbidden, "you must join the room to see messages")
		return
	}

	// Get recent messages (last 100)
	// In a production app, you'd want pagination or infinite scroll
	messages, err := app.store.Messages.GetRoomMessages(r.Context(), roomID, 100)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to retrieve messages")
		return
	}

	// Return empty array instead of null if no messages
	if messages == nil {
		messages = []*store.Message{}
	}

	writeJSON(w, http.StatusOK, messages)
}
