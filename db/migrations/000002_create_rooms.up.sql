-- Create rooms table for chat rooms
-- Each room is a separate chat channel that users can join
CREATE TABLE IF NOT EXISTS rooms (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    created_by BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Index on name for fast room lookup
CREATE INDEX idx_rooms_name ON rooms(name);

-- Index on created_by to find rooms created by a specific user
CREATE INDEX idx_rooms_created_by ON rooms(created_by);
