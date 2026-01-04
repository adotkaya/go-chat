package main

import (
	"log"

	"github.com/drazan344/go-chat/internal/db"
	"github.com/drazan344/go-chat/internal/env"
	"github.com/drazan344/go-chat/internal/store"
	"github.com/drazan344/go-chat/internal/websocket"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq" // PostgreSQL driver
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found or couldn't be loaded: %v", err)
	}

	cfg := config{
		addr: env.GetString("ADDR", ":8080"),
		db: dbConfig{
			addr:         env.GetString("DB_ADDR", "postgres://user:adminpassword@localhost/social?sslmode=disable"),
			maxOpenConns: env.GetInt("DB_MAX_OPEN_CONNS", 25),
			maxIdleConns: env.GetInt("DB_MAX_IDLE_CONNS", 25),
			maxIdleTime:  env.GetString("DB_MAX_IDLE_TIME", "5m"),
		},
		auth: authConfig{
			jwtSecret: env.GetString("JWT_SECRET", "my-secret-key-change-in-production"),
		},
	}

	// Initialize database connection
	// This creates a connection pool to PostgreSQL with the configured parameters
	database, err := db.New(
		cfg.db.addr,
		cfg.db.maxOpenConns,
		cfg.db.maxIdleConns,
		cfg.db.maxIdleTime,
	)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer database.Close()
	log.Println("Database connection established successfully")

	// Create storage layer with the database connection
	store := store.NewPostgresStorage(database)

	// Create and start WebSocket hub for real-time messaging
	// The hub manages all WebSocket connections and message broadcasting
	hub := websocket.NewHub(store)
	go hub.Run() // Start hub in background goroutine
	log.Println("WebSocket hub initialized and running")

	app := &application{
		config: cfg,
		store:  store,
		hub:    hub,
	}

	// Initialize the application

	mux := app.mount()
	log.Fatal(app.run(mux))
}
