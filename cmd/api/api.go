package main

import (
	"log"
	"net/http"
	"time"

	"github.com/drazan344/go-chat/internal/store"
	"github.com/drazan344/go-chat/internal/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type application struct {
	// Define your application struct fields here
	config config
	store  store.Storage
	hub    *websocket.Hub // WebSocket hub for real-time messaging
}

type config struct {
	// Define your config struct fields here
	addr string
	db   dbConfig
	auth authConfig
}

type dbConfig struct {
	addr         string
	maxOpenConns int
	maxIdleConns int
	maxIdleTime  string
}

type authConfig struct {
	jwtSecret string // Secret key for signing JWT tokens
}

func (app *application) mount() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Set a timeout value on the request context (ctx), that will signal
	// through ctx.Done() that the request has timed out and further
	// processing should be stopped.
	r.Use(middleware.Timeout(60 * time.Second))

	// Serve static files
	fileServer := http.FileServer(http.Dir("./web/static"))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	// Serve index.html at root
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/index.html")
	})

	// API routes
	r.Route("/v1", func(r chi.Router) {
		// Health check endpoint
		r.Get("/health", app.healthCheckHandler)

		// Public authentication routes (no auth required)
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", app.registerHandler)
			r.Post("/login", app.loginHandler)
		})

		// Protected routes (require authentication)
		// The AuthMiddleware validates JWT and adds user ID to context
		r.Group(func(r chi.Router) {
			r.Use(app.AuthMiddleware)

			// Current user endpoint
			r.Get("/auth/me", app.getCurrentUserHandler)

			// Room routes
			r.Route("/rooms", func(r chi.Router) {
				r.Get("/", app.listRoomsHandler)
				r.Post("/", app.createRoomHandler)
				r.Get("/{roomID}", app.getRoomHandler)
				r.Post("/{roomID}/join", app.joinRoomHandler)
				r.Post("/{roomID}/leave", app.leaveRoomHandler)
				r.Get("/{roomID}/messages", app.getRoomMessagesHandler)

				// WebSocket endpoint for real-time chat
				r.Get("/{roomID}/ws", app.websocketHandler)
			})
		})
	})

	return r
}

func (app *application) run(mux http.Handler) error {

	srv := &http.Server{
		Addr:         app.config.addr,
		Handler:      mux,
		WriteTimeout: time.Second * 30,
		ReadTimeout:  time.Second * 10,
		IdleTimeout:  time.Minute,
	}

	log.Printf("Server has started at %s", app.config.addr)

	return srv.ListenAndServe()
}
