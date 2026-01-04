package store

import (
	"context"
	"database/sql"
	"time"
)

// RoomMember represents the many-to-many relationship between users and rooms
// This tracks which users have joined which rooms
type RoomMember struct {
	RoomID   int64     `json:"room_id"`
	UserID   int64     `json:"user_id"`
	JoinedAt time.Time `json:"joined_at"`
}

// RoomMemberStore handles database operations for room memberships
type RoomMemberStore struct {
	db *sql.DB
}

// Join adds a user to a room
// If the user is already a member, this will return an error due to the primary key constraint
func (s *RoomMemberStore) Join(ctx context.Context, roomID, userID int64) error {
	query := `
		INSERT INTO room_members (room_id, user_id)
		VALUES ($1, $2)
	`

	// ExecContext is used when we don't need to retrieve any data back
	// It's more efficient than QueryRowContext for INSERT/UPDATE/DELETE without RETURNING
	_, err := s.db.ExecContext(ctx, query, roomID, userID)
	return err
}

// Leave removes a user from a room
// If the user is not a member, this will not return an error (idempotent operation)
func (s *RoomMemberStore) Leave(ctx context.Context, roomID, userID int64) error {
	query := `
		DELETE FROM room_members
		WHERE room_id = $1 AND user_id = $2
	`

	_, err := s.db.ExecContext(ctx, query, roomID, userID)
	return err
}

// IsUserInRoom checks if a user is a member of a specific room
// This is important for authorization (user can only see messages in rooms they've joined)
func (s *RoomMemberStore) IsUserInRoom(ctx context.Context, roomID, userID int64) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM room_members
			WHERE room_id = $1 AND user_id = $2
		)
	`

	var exists bool
	// EXISTS returns a boolean indicating whether any rows match the condition
	err := s.db.QueryRowContext(ctx, query, roomID, userID).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// GetRoomMembers retrieves all user IDs for members of a specific room
// This can be used to determine who should receive messages in the room
func (s *RoomMemberStore) GetRoomMembers(ctx context.Context, roomID int64) ([]int64, error) {
	query := `
		SELECT user_id
		FROM room_members
		WHERE room_id = $1
		ORDER BY joined_at ASC
	`

	rows, err := s.db.QueryContext(ctx, query, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	userIDs := make([]int64, 0)
	for rows.Next() {
		var userID int64
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		userIDs = append(userIDs, userID)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return userIDs, nil
}

// GetRoomMemberCount returns the number of members in a room
// Useful for displaying room statistics in the UI
func (s *RoomMemberStore) GetRoomMemberCount(ctx context.Context, roomID int64) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM room_members
		WHERE room_id = $1
	`

	var count int
	err := s.db.QueryRowContext(ctx, query, roomID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}
