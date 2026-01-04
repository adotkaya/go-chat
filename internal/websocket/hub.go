package websocket

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/drazan344/go-chat/internal/store"
)

// Message represents a chat message being sent through WebSocket
// This is used for both incoming and outgoing messages
type Message struct {
	RoomID   int64  `json:"room_id"`
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Content  string `json:"content"`
	Type     string `json:"type"` // "message", "join", "leave"
}

// Hub maintains the set of active clients and broadcasts messages to clients
// It's the central coordinator for all WebSocket connections
type Hub struct {
	// Registered clients organized by room ID
	// map[roomID]map[*Client]bool
	// The inner map acts as a set (we only care about keys, values are always true)
	rooms map[int64]map[*Client]bool

	// Inbound messages from the clients
	// Messages are sent to this channel from client.readPump()
	broadcast chan *Message

	// Register requests from the clients
	// Sent when a new WebSocket connection is established
	register chan *Client

	// Unregister requests from clients
	// Sent when a WebSocket connection is closed
	unregister chan *Client

	// Storage layer for persisting messages
	store store.Storage
}

// NewHub creates a new Hub instance
// The hub must be started with hub.Run() in a goroutine
func NewHub(store store.Storage) *Hub {
	return &Hub{
		broadcast:  make(chan *Message, 256), // Buffered to prevent blocking
		register:   make(chan *Client),
		unregister: make(chan *Client),
		rooms:      make(map[int64]map[*Client]bool),
		store:      store,
	}
}

// Run starts the hub's main event loop
// This should be called in a goroutine: go hub.Run()
// The hub continuously listens on its channels and processes events
func (h *Hub) Run() {
	log.Println("WebSocket hub started")

	for {
		select {
		case client := <-h.register:
			// A new client wants to connect to a room
			h.registerClient(client)

		case client := <-h.unregister:
			// A client disconnected from a room
			h.unregisterClient(client)

		case message := <-h.broadcast:
			// A message needs to be broadcasted to all clients in a room
			h.handleBroadcast(message)
		}
	}
}

// registerClient adds a client to a room
func (h *Hub) registerClient(client *Client) {
	// Check if room exists in the map
	if h.rooms[client.roomID] == nil {
		// Create a new set for this room
		h.rooms[client.roomID] = make(map[*Client]bool)
	}

	// Add client to the room
	h.rooms[client.roomID][client] = true

	log.Printf("Client registered: user=%d room=%d (total in room: %d)",
		client.userID, client.roomID, len(h.rooms[client.roomID]))

	// Optionally send a "user joined" notification to the room
	joinMessage := &Message{
		RoomID:   client.roomID,
		UserID:   client.userID,
		Username: client.username,
		Content:  client.username + " joined the room",
		Type:     "join",
	}

	// Broadcast join message to all clients in the room
	h.broadcastToRoom(client.roomID, joinMessage)
}

// unregisterClient removes a client from a room
func (h *Hub) unregisterClient(client *Client) {
	if clients, ok := h.rooms[client.roomID]; ok {
		if _, ok := clients[client]; ok {
			// Remove client from room
			delete(clients, client)

			// Close the client's send channel
			close(client.send)

			log.Printf("Client unregistered: user=%d room=%d (remaining in room: %d)",
				client.userID, client.roomID, len(clients))

			// If room is empty, delete it from the map
			if len(clients) == 0 {
				delete(h.rooms, client.roomID)
				log.Printf("Room %d is now empty and removed from hub", client.roomID)
			}

			// Send a "user left" notification
			leaveMessage := &Message{
				RoomID:   client.roomID,
				UserID:   client.userID,
				Username: client.username,
				Content:  client.username + " left the room",
				Type:     "leave",
			}

			// Broadcast leave message to remaining clients
			h.broadcastToRoom(client.roomID, leaveMessage)
		}
	}
}

// handleBroadcast processes incoming messages
// It persists the message to the database and broadcasts it to all clients in the room
func (h *Hub) handleBroadcast(message *Message) {
	// Only persist actual chat messages, not join/leave notifications
	if message.Type == "message" {
		// Save message to database
		// Using context.Background() since this is not tied to a specific HTTP request
		// In production, you might want a context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		dbMessage := &store.Message{
			RoomID:  message.RoomID,
			UserID:  message.UserID,
			Content: message.Content,
		}

		if err := h.store.Messages.Create(ctx, dbMessage); err != nil {
			log.Printf("Failed to save message to database: %v", err)
			// Continue with broadcast even if database save fails
			// In production, you might want to handle this differently
		}
	}

	// Broadcast message to all clients in the room
	h.broadcastToRoom(message.RoomID, message)
}

// broadcastToRoom sends a message to all clients in a specific room
// This is a fan-out pattern: one message goes to many recipients
func (h *Hub) broadcastToRoom(roomID int64, message *Message) {
	// Get all clients in the room
	clients, ok := h.rooms[roomID]
	if !ok {
		// No clients in this room
		return
	}

	// Marshal message to JSON
	// We do this once instead of for each client (more efficient)
	jsonMessage, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal message: %v", err)
		return
	}

	// Send message to each client in the room
	// This is the fan-out: iterate through all clients and send to each
	for client := range clients {
		select {
		case client.send <- jsonMessage:
			// Message sent successfully
			// The non-blocking select prevents one slow client from blocking others
		default:
			// Client's send buffer is full, likely disconnected
			// Close and unregister the client
			close(client.send)
			delete(clients, client)
			log.Printf("Client removed due to full buffer: user=%d room=%d", client.userID, roomID)
		}
	}

	log.Printf("Broadcasted message to %d clients in room %d", len(clients), roomID)
}

// GetRoomClientCount returns the number of active clients in a room
// This can be used for monitoring or displaying "X users online" in UI
func (h *Hub) GetRoomClientCount(roomID int64) int {
	if clients, ok := h.rooms[roomID]; ok {
		return len(clients)
	}
	return 0
}
