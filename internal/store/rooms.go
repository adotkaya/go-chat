package store

import (
	"context"
	"database/sql"
	"time"
)

// Room represents a chat room where users can send messages
// Rooms are created by users and can be joined by other users
type Room struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedBy   int64     `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// RoomStore handles database operations for rooms
// It follows the repository pattern for clean separation of data access logic
type RoomStore struct {
	db *sql.DB
}

// Create creates a new chat room in the database
// It returns the generated ID and timestamps via the RETURNING clause
func (s *RoomStore) Create(ctx context.Context, room *Room) error {
	query := `
		INSERT INTO rooms (name, description, created_by)
		VALUES ($1, $2, $3) RETURNING id, created_at, updated_at
	`

	// QueryRowContext executes the query and scans the result in one operation
	// Context allows for timeout and cancellation
	err := s.db.QueryRowContext(
		ctx,
		query,
		room.Name,
		room.Description,
		room.CreatedBy,
	).Scan(
		&room.ID,
		&room.CreatedAt,
		&room.UpdatedAt,
	)
	if err != nil {
		return err
	}
	return nil
}

// GetByID retrieves a room by its ID
func (s *RoomStore) GetByID(ctx context.Context, id int64) (*Room, error) {
	query := `
		SELECT id, name, description, created_by, created_at, updated_at
		FROM rooms
		WHERE id = $1
	`

	room := &Room{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&room.ID,
		&room.Name,
		&room.Description,
		&room.CreatedBy,
		&room.CreatedAt,
		&room.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return room, nil
}

// GetByName retrieves a room by its name
// Room names are unique, so this will return at most one room
func (s *RoomStore) GetByName(ctx context.Context, name string) (*Room, error) {
	query := `
		SELECT id, name, description, created_by, created_at, updated_at
		FROM rooms
		WHERE name = $1
	`

	room := &Room{}
	err := s.db.QueryRowContext(ctx, query, name).Scan(
		&room.ID,
		&room.Name,
		&room.Description,
		&room.CreatedBy,
		&room.CreatedAt,
		&room.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return room, nil
}

// List retrieves all rooms from the database
// Returns rooms ordered by creation time (newest first)
func (s *RoomStore) List(ctx context.Context) ([]*Room, error) {
	query := `
		SELECT id, name, description, created_by, created_at, updated_at
		FROM rooms
		ORDER BY created_at DESC
	`

	// Query returns multiple rows, unlike QueryRow
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close() // Important: always close rows to free resources

	rooms := make([]*Room, 0)
	// Iterate through each row in the result set
	for rows.Next() {
		room := &Room{}
		err := rows.Scan(
			&room.ID,
			&room.Name,
			&room.Description,
			&room.CreatedBy,
			&room.CreatedAt,
			&room.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		rooms = append(rooms, room)
	}

	// Check for errors that occurred during iteration
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return rooms, nil
}

// GetUserRooms retrieves all rooms that a user has joined
// This joins the rooms and room_members tables
func (s *RoomStore) GetUserRooms(ctx context.Context, userID int64) ([]*Room, error) {
	query := `
		SELECT r.id, r.name, r.description, r.created_by, r.created_at, r.updated_at
		FROM rooms r
		INNER JOIN room_members rm ON r.id = rm.room_id
		WHERE rm.user_id = $1
		ORDER BY r.created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rooms := make([]*Room, 0)
	for rows.Next() {
		room := &Room{}
		err := rows.Scan(
			&room.ID,
			&room.Name,
			&room.Description,
			&room.CreatedBy,
			&room.CreatedAt,
			&room.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		rooms = append(rooms, room)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return rooms, nil
}

// Delete deletes a room by its ID
// CASCADE will automatically delete related messages and room_members
func (s *RoomStore) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM rooms WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}
