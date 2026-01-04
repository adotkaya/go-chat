package store

import (
	"context"
	"database/sql"
	"time"
)

// Message represents a chat message in a room
// Messages are persisted to the database for history and reliability
type Message struct {
	ID        int64     `json:"id"`
	RoomID    int64     `json:"room_id"`
	UserID    int64     `json:"user_id"`
	Content   string    `json:"content"`
	Username  string    `json:"username"`  // Joined from users table for display purposes
	CreatedAt time.Time `json:"created_at"`
}

// MessageStore handles database operations for messages
type MessageStore struct {
	db *sql.DB
}

// Create inserts a new message into the database
// The message must belong to a room and be sent by a user
func (s *MessageStore) Create(ctx context.Context, message *Message) error {
	query := `
		INSERT INTO messages (room_id, user_id, content)
		VALUES ($1, $2, $3) RETURNING id, created_at
	`

	err := s.db.QueryRowContext(
		ctx,
		query,
		message.RoomID,
		message.UserID,
		message.Content,
	).Scan(
		&message.ID,
		&message.CreatedAt,
	)
	if err != nil {
		return err
	}
	return nil
}

// GetRoomMessages retrieves the most recent messages for a room
// Messages are joined with the users table to include the username
// The limit parameter controls how many messages to return (e.g., last 100 messages)
func (s *MessageStore) GetRoomMessages(ctx context.Context, roomID int64, limit int) ([]*Message, error) {
	// Join with users table to get username for display
	// Order by created_at DESC and then reverse in code, or use a subquery
	query := `
		SELECT m.id, m.room_id, m.user_id, m.content, u.username, m.created_at
		FROM messages m
		INNER JOIN users u ON m.user_id = u.id
		WHERE m.room_id = $1
		ORDER BY m.created_at DESC
		LIMIT $2
	`

	rows, err := s.db.QueryContext(ctx, query, roomID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Store messages in reverse order since we queried DESC but want to display ASC
	messages := make([]*Message, 0, limit)
	for rows.Next() {
		message := &Message{}
		err := rows.Scan(
			&message.ID,
			&message.RoomID,
			&message.UserID,
			&message.Content,
			&message.Username,
			&message.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	// Reverse the slice to get chronological order (oldest to newest)
	// This makes it easier to display in the UI
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// GetMessagesSince retrieves all messages in a room since a specific timestamp
// This is useful for clients that reconnect and want to catch up on missed messages
func (s *MessageStore) GetMessagesSince(ctx context.Context, roomID int64, since time.Time) ([]*Message, error) {
	query := `
		SELECT m.id, m.room_id, m.user_id, m.content, u.username, m.created_at
		FROM messages m
		INNER JOIN users u ON m.user_id = u.id
		WHERE m.room_id = $1 AND m.created_at > $2
		ORDER BY m.created_at ASC
	`

	rows, err := s.db.QueryContext(ctx, query, roomID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := make([]*Message, 0)
	for rows.Next() {
		message := &Message{}
		err := rows.Scan(
			&message.ID,
			&message.RoomID,
			&message.UserID,
			&message.Content,
			&message.Username,
			&message.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}
