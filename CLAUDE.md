# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Run Commands

**Build the application:**
```bash
go build -o bin/chat cmd/api/*.go
# Or: make build
```

**Run the application:**
```bash
go run cmd/api/*.go
# Or: make run
```

**Run database migrations:**
```bash
go run cmd/migrate/main.go up
# Or: make migrate-up
```

**Rollback last migration:**
```bash
go run cmd/migrate/main.go down
# Or: make migrate-down
```

**Install/update dependencies:**
```bash
go mod tidy
# Or: make deps
```

**Complete setup:**
```bash
make setup  # Runs deps and migrate-up
```

## Architecture Overview

This is a real-time chat application with WebSocket-based messaging, JWT authentication, and a terminal-style UI. The application demonstrates Go backend principles suitable for junior developers.

### Core Structure

**cmd/api/** - HTTP server and handlers
- `main.go` - Application entry point, initializes database, store, WebSocket hub
- `api.go` - Router configuration, application struct, static file serving
- `auth.go` - Registration, login, and current user handlers
- `rooms.go` - Room CRUD, join/leave, message history handlers
- `websocket.go` - WebSocket upgrade and connection handling
- `middleware.go` - JWT authentication middleware
- `helpers.go` - JSON helpers, error responses, URL parameter extraction
- `health.go` - Health check endpoint

**cmd/migrate/** - Database migration tool
- `main.go` - Custom migration runner supporting up/down commands

**internal/auth/** - Authentication package
- `auth.go` - Password hashing (bcrypt), JWT generation/validation

**internal/db/** - Database connection management
- `db.go` - PostgreSQL connection with pooling configuration

**internal/store/** - Data access layer (Repository pattern)
- `storage.go` - Storage interface aggregating all stores
- `users.go` - User model and UserStore (Create, GetByEmail, GetByID)
- `rooms.go` - Room model and RoomStore (Create, GetByID, GetByName, List, GetUserRooms, Delete)
- `messages.go` - Message model and MessageStore (Create, GetRoomMessages, GetMessagesSince)
- `room_members.go` - RoomMember model and RoomMemberStore (Join, Leave, IsUserInRoom, GetRoomMembers)
- All stores use `context.Context` for timeout/cancellation support

**internal/websocket/** - Real-time messaging (Hub pattern)
- `hub.go` - Central hub managing all WebSocket clients, message broadcasting, room management
- `client.go` - Individual WebSocket client with readPump/writePump goroutines

**web/** - Frontend files
- `index.html` - Single-page application structure
- `static/css/style.css` - Terminal-style CSS theme
- `static/js/auth.js` - Authentication client
- `static/js/websocket.js` - WebSocket client wrapper
- `static/js/chat.js` - Chat application logic
- `static/js/app.js` - Application initialization

### Key Design Patterns

**Repository Pattern:** Data access abstracted through store interfaces in `internal/store/storage.go`. Each model has its own store with methods for database operations.

**WebSocket Hub Pattern:** Central hub (`internal/websocket/hub.go`) manages all clients and broadcasts messages. Clients register/unregister via channels. Messages flow through channels for thread-safe communication.

**Middleware Chain:** Chi router middleware stack includes RequestID, RealIP, Logger, Recoverer, and Timeout. Authentication middleware validates JWT and adds user ID to context.

**Dependency Injection:** The `application` struct holds config, store, and hub. All handlers are methods on this struct, accessing dependencies without globals.

**Context Usage:** Request context flows through all layers. Used for authentication (storing user ID), database operations (timeouts), and request lifecycle management.

## Database

**Schema (4 migrations in db/migrations/):**

1. `users` - id, username (unique), email (unique), password (bcrypt), created_at, updated_at
2. `rooms` - id, name (unique), description, created_by (FK to users), created_at, updated_at
3. `messages` - id, room_id (FK to rooms), user_id (FK to users), content, created_at
4. `room_members` - room_id (FK), user_id (FK), joined_at (composite PK)

**Connection:**
```
postgres://user:password@localhost:5432/gochat?sslmode=disable
```

**Environment variables:**
- `DB_ADDR` - Connection string
- `DB_MAX_OPEN_CONNS` - Max open connections (default: 25)
- `DB_MAX_IDLE_CONNS` - Max idle connections (default: 25)
- `DB_MAX_IDLE_TIME` - Max idle time (default: "5m")

## Authentication

**JWT Configuration:**
- Secret: `JWT_SECRET` environment variable
- Expiration: 24 hours
- Algorithm: HMAC-SHA256
- Claims: user_id, iat, exp, issuer

**Password Hashing:**
- bcrypt with DefaultCost (10)
- Never store plain text passwords

**Auth Flow:**
1. User registers/logins → receives JWT token
2. Client stores token in localStorage
3. Subsequent requests include `Authorization: Bearer <token>` header
4. AuthMiddleware validates token and adds user ID to context
5. Handlers extract user ID with `GetUserIDFromContext()`

## WebSocket Flow

**Connection:**
1. Client connects to `/v1/rooms/{roomID}/ws` with JWT token
2. Handler verifies user is room member
3. HTTP upgraded to WebSocket
4. Client instance created with send channel (buffered to 256)
5. Client registered with hub
6. readPump and writePump goroutines started

**Message Broadcasting:**
1. readPump receives message from WebSocket
2. Message sent to hub.broadcast channel
3. Hub persists message to database
4. Hub marshals message to JSON (once)
5. Hub broadcasts to all clients in room via their send channels
6. writePump sends from send channel to WebSocket

**Disconnection:**
1. WebSocket error/close detected in readPump
2. Client sent to hub.unregister channel
3. Hub removes client from room
4. Send channel closed
5. Join/leave notifications sent to room

## API Endpoints

**Public:**
- `POST /v1/auth/register` - Register (username, email, password)
- `POST /v1/auth/login` - Login (email, password)

**Protected (require Authorization header):**
- `GET /v1/auth/me` - Get current user
- `GET /v1/rooms` - List all rooms
- `POST /v1/rooms` - Create room (auto-joins creator)
- `GET /v1/rooms/{id}` - Get room details
- `POST /v1/rooms/{id}/join` - Join room
- `POST /v1/rooms/{id}/leave` - Leave room
- `GET /v1/rooms/{id}/messages` - Get message history (requires membership)
- `GET /v1/rooms/{id}/ws` - WebSocket connection (requires membership)

## Frontend

**Terminal Style:**
- Colors: black background (#0a0a0a), green text (#00ff00), amber highlights (#ffb000)
- Monospace font: Courier New
- Box shadows with green glow for terminal windows
- Custom scrollbar styled to match theme

**JavaScript Classes:**
- `Auth` - Handles registration, login, token storage in localStorage
- `WebSocketClient` - Manages WebSocket connection with auto-reconnect
- `ChatApp` - Main application logic: room management, message display, WebSocket integration
- `app.js` - Initialization, event listeners, modal management

**State Management:**
- JWT token and user stored in localStorage
- Current room tracked in ChatApp instance
- WebSocket client recreated when switching rooms
- Message history loaded on room join

## Common Development Tasks

**Adding a new API endpoint:**
1. Create handler function in appropriate file (e.g., `cmd/api/rooms.go`)
2. Add route in `cmd/api/api.go` mount() function
3. Use helper functions: `writeJSON()`, `readJSON()`, `writeError()`, `extractIDFromURL()`
4. Extract user ID with `GetUserIDFromContext()` if protected

**Adding a database table:**
1. Create migration files in `db/migrations/` (XXXXXX_name.up.sql and .down.sql)
2. Create model struct in `internal/store/`
3. Create store struct with methods
4. Add store interface to `Storage` struct in `storage.go`
5. Initialize store in `NewPostgresStorage()`

**Modifying WebSocket messages:**
1. Update `Message` struct in `internal/websocket/hub.go`
2. Update message handling in hub.handleBroadcast()
3. Update frontend message display in `web/static/js/chat.js` displayMessage()

## Educational Notes

**For Junior Developers:**

This codebase demonstrates:
- Go project structure and package organization
- HTTP server with Chi router and middleware
- PostgreSQL integration with connection pooling
- Repository pattern for data access
- JWT authentication and bcrypt password hashing
- WebSocket hub pattern for real-time communication
- Goroutines and channels for concurrency
- Context usage throughout the stack
- Frontend integration without frameworks

**Code Comments:**
- All packages have educational comments explaining Go concepts
- Security considerations documented (bcrypt, JWT, SQL injection prevention)
- Concurrency patterns explained (goroutines, channels, fan-out)
- Best practices highlighted throughout

## Environment Setup

1. Install Go 1.24+
2. Install PostgreSQL
3. Create database: `createdb gochat`
4. Copy `.env.example` to `.env` and configure
5. Run migrations: `make migrate-up`
6. Start server: `make run`
7. Open browser to http://localhost:8080

## Server Configuration

Default address: `:8080` (configurable via `ADDR` environment variable)

Server timeouts:
- Write: 30 seconds
- Read: 10 seconds
- Idle: 1 minute
- Request: 60 seconds (middleware timeout)

## Dependencies

- `github.com/go-chi/chi/v5` v5.2.2 - HTTP router and middleware
- `github.com/joho/godotenv` v1.5.1 - Environment file loading
- `github.com/lib/pq` v1.10.9 - PostgreSQL driver
- `golang.org/x/crypto` - bcrypt password hashing
- `github.com/golang-jwt/jwt/v5` - JWT token handling
- `github.com/gorilla/websocket` - WebSocket implementation
