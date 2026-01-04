package store

import (
	"context"
	"database/sql"
	"time"
)

// Storage aggregates all store interfaces
// This follows the repository pattern, providing a clean abstraction over data access
type Storage struct {
	// Posts store (kept for compatibility, though not used in chat app)
	Posts interface {
		Create(context.Context, *Post) error
	}

	// Users store handles user account management
	Users interface {
		Create(context.Context, *User) error
		GetByEmail(context.Context, string) (*User, error)
		GetByID(context.Context, int64) (*User, error)
	}

	// Rooms store handles chat room management
	Rooms interface {
		Create(context.Context, *Room) error
		GetByID(context.Context, int64) (*Room, error)
		GetByName(context.Context, string) (*Room, error)
		List(context.Context) ([]*Room, error)
		GetUserRooms(context.Context, int64) ([]*Room, error)
		Delete(context.Context, int64) error
	}

	// Messages store handles chat message persistence
	Messages interface {
		Create(context.Context, *Message) error
		GetRoomMessages(context.Context, int64, int) ([]*Message, error)
		GetMessagesSince(context.Context, int64, time.Time) ([]*Message, error)
	}

	// RoomMembers store handles room membership (many-to-many user-room relationship)
	RoomMembers interface {
		Join(context.Context, int64, int64) error
		Leave(context.Context, int64, int64) error
		IsUserInRoom(context.Context, int64, int64) (bool, error)
		GetRoomMembers(context.Context, int64) ([]int64, error)
		GetRoomMemberCount(context.Context, int64) (int, error)
	}
}

// NewPostgresStorage creates a new Storage instance with PostgreSQL implementations
// All stores share the same database connection pool for efficiency
func NewPostgresStorage(db *sql.DB) Storage {
	return Storage{
		Posts:       &PostStore{db},
		Users:       &UserStore{db},
		Rooms:       &RoomStore{db},
		Messages:    &MessageStore{db},
		RoomMembers: &RoomMemberStore{db},
	}
}
