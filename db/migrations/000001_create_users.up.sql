-- Create users table for authentication and user management
-- This table stores user account information including credentials
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,  -- Stores bcrypt hashed password (never plain text!)
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Index on email for fast lookup during login
CREATE INDEX idx_users_email ON users(email);

-- Index on username for fast lookup and uniqueness checks
CREATE INDEX idx_users_username ON users(username);
