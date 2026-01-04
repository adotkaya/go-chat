# Go Chat Application

A real-time chat application built with Go, featuring WebSocket-based messaging, JWT authentication, and a retro terminal-style UI.

## Features

- **Real-time Messaging**: WebSocket-based chat with instant message delivery
- **Public Chat Rooms**: Create and join named chat rooms (like Slack/Discord)
- **Full Authentication**: Email/password registration and login with JWT tokens
- **Terminal-Style UI**: Cool retro aesthetic with green/amber terminal theme
- **Message Persistence**: All messages saved to PostgreSQL database
- **Room Management**: Create rooms, join/leave rooms, view room lists
- **Educational Code**: Well-commented, clean code suitable for learning

## Tech Stack

**Backend:**
- Go 1.24.5
- Chi Router v5 - HTTP routing and middleware
- PostgreSQL - Data persistence
- WebSockets (gorilla/websocket) - Real-time communication
- JWT (golang-jwt/jwt) - Authentication tokens
- bcrypt - Password hashing

**Frontend:**
- Vanilla JavaScript (no frameworks)
- Pure CSS (terminal theme)
- WebSocket API

## Prerequisites

**With Docker (Recommended - Easiest):**
- Docker Desktop or Docker Engine
- docker-compose

**Without Docker:**
- Go 1.24 or higher
- PostgreSQL 12 or higher
- Make (optional, for convenience commands)

## Quick Start

### Option 1: Using Docker (Recommended) 🐳

The easiest way to run the application - everything is configured automatically!

```bash
# 1. Clone the repository
git clone https://github.com/drazan344/go-chat.git
cd go-chat

# 2. Start everything with one command
docker-compose up

# That's it! Open http://localhost:8080 in your browser
```

**What Docker does for you:**
- ✅ Automatically sets up PostgreSQL database
- ✅ Runs database migrations
- ✅ Builds and starts the Go application
- ✅ No need to install Go or PostgreSQL locally
- ✅ Consistent environment across all machines

**Useful Docker commands:**
```bash
# Start in detached mode (background)
docker-compose up -d

# View logs
docker-compose logs -f app

# Stop everything
docker-compose down

# Rebuild after code changes
docker-compose up --build

# Remove all data (fresh start)
docker-compose down -v
```

### Option 2: Manual Setup (Without Docker)

### 1. Clone and Install Dependencies

```bash
git clone https://github.com/drazan344/go-chat.git
cd go-chat
go mod tidy
```

### 2. Setup Database

Create a PostgreSQL database:

```bash
createdb gochat
```

Copy the example environment file and configure your database:

```bash
cp .env.example .env
# Edit .env and update DB_ADDR with your database credentials
```

### 3. Run Migrations

```bash
make migrate-up
# Or: go run cmd/migrate/main.go up
```

### 4. Start the Server

```bash
make run
# Or: go run cmd/api/*.go
```

The server will start on http://localhost:8080

### 5. Open in Browser

Navigate to http://localhost:8080 and create an account to start chatting!

## Project Structure

```
go-chat/
├── cmd/
│   ├── api/              # HTTP server and handlers
│   │   ├── main.go       # Application entry point
│   │   ├── api.go        # Router and application struct
│   │   ├── auth.go       # Authentication handlers
│   │   ├── rooms.go      # Room management handlers
│   │   ├── websocket.go  # WebSocket handler
│   │   ├── middleware.go # Auth middleware
│   │   ├── helpers.go    # Helper functions
│   │   └── health.go     # Health check
│   └── migrate/          # Database migration tool
│       └── main.go
├── internal/
│   ├── auth/             # Authentication package
│   │   └── auth.go       # JWT and bcrypt functions
│   ├── db/               # Database connection
│   │   └── db.go
│   ├── env/              # Environment utilities
│   │   └── env.go
│   ├── store/            # Data access layer
│   │   ├── storage.go    # Storage interface
│   │   ├── users.go      # User model and store
│   │   ├── rooms.go      # Room model and store
│   │   ├── messages.go   # Message model and store
│   │   └── room_members.go
│   └── websocket/        # WebSocket hub pattern
│       ├── hub.go        # Message broadcasting hub
│       └── client.go     # WebSocket client
├── db/migrations/        # SQL migration files
├── web/                  # Frontend files
│   ├── index.html
│   └── static/
│       ├── css/
│       │   └── style.css
│       └── js/
│           ├── auth.js
│           ├── websocket.js
│           ├── chat.js
│           └── app.js
├── .env.example          # Example environment variables
├── Makefile              # Build and run commands
├── CLAUDE.md             # Development guide
└── README.md
```

## API Endpoints

### Authentication (Public)
- `POST /v1/auth/register` - Register new user
- `POST /v1/auth/login` - Login and receive JWT token

### Authentication (Protected)
- `GET /v1/auth/me` - Get current user info

### Rooms (Protected)
- `GET /v1/rooms` - List all rooms
- `POST /v1/rooms` - Create new room
- `GET /v1/rooms/{id}` - Get room details
- `POST /v1/rooms/{id}/join` - Join a room
- `POST /v1/rooms/{id}/leave` - Leave a room
- `GET /v1/rooms/{id}/messages` - Get room message history

### WebSocket (Protected)
- `GET /v1/rooms/{id}/ws` - WebSocket connection for real-time chat

## Makefile Commands

```bash
make build       # Build the application
make run         # Run the application
make migrate-up  # Run database migrations
make migrate-down # Rollback last migration
make test        # Run tests
make clean       # Clean build artifacts
make deps        # Install dependencies
make setup       # Complete development setup
```

## Environment Variables

See `.env.example` for all available configuration options.

## Development

This project is designed to be educational and readable. Key concepts demonstrated:

- **Repository Pattern**: Clean separation of data access in `internal/store/`
- **Middleware Chain**: Authentication and logging middleware
- **WebSocket Hub Pattern**: Centralized message broadcasting
- **Context Usage**: Request-scoped values and cancellation
- **JWT Authentication**: Stateless token-based auth
- **bcrypt**: Secure password hashing
- **Goroutines & Channels**: Concurrent WebSocket handling

## Security Notes

- Passwords are hashed with bcrypt before storage
- JWT tokens expire after 24 hours
- SQL injection prevented through parameterized queries
- WebSocket connections require authentication

## Future Enhancements

- User presence indicators
- Typing indicators
- File uploads
- Direct messages
- User profiles
- Message editing/deletion
- Search functionality

## License

MIT

## Author

Built as an educational project to demonstrate Go backend development and real-time web applications.
