-- Create room_members table for tracking room membership
-- This is a many-to-many join table between users and rooms
-- It tracks which users have joined which rooms
CREATE TABLE IF NOT EXISTS room_members (
    room_id BIGINT NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    joined_at TIMESTAMP NOT NULL DEFAULT NOW(),
    -- Composite primary key ensures a user can only join a room once
    PRIMARY KEY (room_id, user_id)
);

-- Index on user_id to find all rooms a user has joined
CREATE INDEX idx_room_members_user_id ON room_members(user_id);

-- Index on room_id to find all members of a room (already covered by PK, but explicit for clarity)
CREATE INDEX idx_room_members_room_id ON room_members(room_id);
