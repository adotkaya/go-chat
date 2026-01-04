package websocket

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	// This should be longer than pongWait to allow for network latency
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait
	// This helps detect broken connections
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer (1MB)
	maxMessageSize = 1024 * 1024
)

// Client represents a single WebSocket connection
// Each user connected to a room has their own Client instance
type Client struct {
	// The WebSocket hub that manages all clients
	hub *Hub

	// The WebSocket connection
	conn *websocket.Conn

	// Buffered channel of outbound messages
	// Using a buffered channel prevents blocking when sending messages
	send chan []byte

	// User information
	userID   int64
	username string

	// Room ID this client is connected to
	roomID int64
}

// readPump pumps messages from the WebSocket connection to the hub
// The application runs readPump in a per-connection goroutine
// This ensures that there is at most one reader on a connection
func (c *Client) readPump() {
	// Cleanup when this function exits
	defer func() {
		// Unregister the client from the hub
		c.hub.unregister <- c
		// Close the WebSocket connection
		c.conn.Close()
	}()

	// Configure connection settings
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))

	// SetPongHandler sets up a handler for pong messages
	// When a pong is received, extend the read deadline
	// This is part of the ping/pong mechanism to detect broken connections
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// Continuously read messages from the WebSocket
	for {
		// ReadMessage blocks until a message is received
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			// WebSocket connection errors are normal when clients disconnect
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Create a message struct to send to the hub
		msg := &Message{
			RoomID:   c.roomID,
			UserID:   c.userID,
			Username: c.username,
			Content:  string(message),
			Type:     "message",
		}

		// Send message to the hub for broadcasting
		// The hub will persist it to the database and broadcast to all clients in the room
		c.hub.broadcast <- msg
	}
}

// writePump pumps messages from the hub to the WebSocket connection
// A goroutine running writePump is started for each connection
// The application ensures that there is at most one writer to a connection
func (c *Client) writePump() {
	// Create a ticker to send ping messages periodically
	// Pings help detect broken connections
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			// Set write deadline
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))

			// Check if channel was closed
			if !ok {
				// The hub closed the channel, close the connection
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Get a writer for the next message
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			// Write the message
			w.Write(message)

			// Add queued messages to the current WebSocket message
			// This is an optimization to batch multiple messages into one WebSocket frame
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			// Close the writer, sending the message
			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			// Send a ping message to the client
			// If the client doesn't respond with a pong, the connection will timeout
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
