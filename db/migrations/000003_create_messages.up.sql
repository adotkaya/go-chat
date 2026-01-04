-- Create messages table for storing chat messages
-- Messages are persisted to database for history and reliability
CREATE TABLE IF NOT EXISTS messages (
    id BIGSERIAL PRIMARY KEY,
    room_id BIGINT NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Index on room_id for efficient message retrieval per room
-- This is the most common query pattern (get messages for a specific room)
CREATE INDEX idx_messages_room_id ON messages(room_id);

-- Index on created_at to support ordering messages chronologically
CREATE INDEX idx_messages_created_at ON messages(created_at);

-- Composite index for getting a specific room's messages in order
CREATE INDEX idx_messages_room_created ON messages(room_id, created_at DESC);
