package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/drazan344/go-chat/internal/env"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// Migration represents a single database migration file
type Migration struct {
	Version  string
	Name     string
	UpSQL    string
	DownSQL  string
	FilePath string
}

func main() {
	// Load .env file for database connection string
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Get command (up or down)
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run cmd/migrate/main.go [up|down]")
	}
	command := os.Args[1]

	// Connect to database
	dbAddr := env.GetString("DB_ADDR", "postgres://user:adminpassword@localhost/social?sslmode=disable")
	db, err := sql.Open("postgres", dbAddr)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}
	log.Println("Connected to database successfully")

	// Create schema_migrations table if it doesn't exist
	// This table tracks which migrations have been applied
	if err := createMigrationsTable(db); err != nil {
		log.Fatal("Failed to create migrations table:", err)
	}

	// Read migration files
	migrations, err := readMigrations("db/migrations")
	if err != nil {
		log.Fatal("Failed to read migrations:", err)
	}

	// Execute command
	switch command {
	case "up":
		if err := migrateUp(db, migrations); err != nil {
			log.Fatal("Migration up failed:", err)
		}
		log.Println("Migration up completed successfully")
	case "down":
		if err := migrateDown(db, migrations); err != nil {
			log.Fatal("Migration down failed:", err)
		}
		log.Println("Migration down completed successfully")
	default:
		log.Fatal("Unknown command. Use 'up' or 'down'")
	}
}

// createMigrationsTable creates the schema_migrations table if it doesn't exist
// This table keeps track of which migrations have been applied to the database
func createMigrationsTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`
	_, err := db.Exec(query)
	return err
}

// readMigrations reads all migration files from the specified directory
// Migration files should follow the naming convention: XXXXXX_name.up.sql and XXXXXX_name.down.sql
func readMigrations(dir string) ([]Migration, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.up.sql"))
	if err != nil {
		return nil, err
	}

	migrations := make([]Migration, 0)
	for _, upFile := range files {
		// Extract version and name from filename
		// Example: 000001_create_users.up.sql -> version: 000001, name: create_users
		baseName := filepath.Base(upFile)
		baseName = strings.TrimSuffix(baseName, ".up.sql")
		parts := strings.SplitN(baseName, "_", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid migration filename: %s", upFile)
		}
		version := parts[0]
		name := parts[1]

		// Read up migration SQL
		upSQL, err := os.ReadFile(upFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", upFile, err)
		}

		// Read down migration SQL
		downFile := filepath.Join(dir, fmt.Sprintf("%s_%s.down.sql", version, name))
		downSQL, err := os.ReadFile(downFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", downFile, err)
		}

		migrations = append(migrations, Migration{
			Version:  version,
			Name:     name,
			UpSQL:    string(upSQL),
			DownSQL:  string(downSQL),
			FilePath: upFile,
		})
	}

	// Sort migrations by version to ensure they're applied in order
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// migrateUp applies all pending migrations
func migrateUp(db *sql.DB, migrations []Migration) error {
	// Get already applied migrations
	applied, err := getAppliedMigrations(db)
	if err != nil {
		return err
	}

	// Apply each migration that hasn't been applied yet
	for _, migration := range migrations {
		if applied[migration.Version] {
			log.Printf("Skipping migration %s_%s (already applied)", migration.Version, migration.Name)
			continue
		}

		log.Printf("Applying migration %s_%s...", migration.Version, migration.Name)

		// Execute the migration in a transaction
		// This ensures that if the migration fails, changes are rolled back
		tx, err := db.Begin()
		if err != nil {
			return err
		}

		// Execute the migration SQL
		if _, err := tx.Exec(migration.UpSQL); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute migration %s: %w", migration.Version, err)
		}

		// Record that this migration was applied
		if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", migration.Version); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %s: %w", migration.Version, err)
		}

		// Commit the transaction
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", migration.Version, err)
		}

		log.Printf("Migration %s_%s applied successfully", migration.Version, migration.Name)
	}

	return nil
}

// migrateDown rolls back the most recent migration
func migrateDown(db *sql.DB, migrations []Migration) error {
	// Get already applied migrations
	applied, err := getAppliedMigrations(db)
	if err != nil {
		return err
	}

	// Find the most recent applied migration
	var lastMigration *Migration
	for i := len(migrations) - 1; i >= 0; i-- {
		if applied[migrations[i].Version] {
			lastMigration = &migrations[i]
			break
		}
	}

	if lastMigration == nil {
		log.Println("No migrations to roll back")
		return nil
	}

	log.Printf("Rolling back migration %s_%s...", lastMigration.Version, lastMigration.Name)

	// Execute the rollback in a transaction
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	// Execute the down migration SQL
	if _, err := tx.Exec(lastMigration.DownSQL); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to execute down migration %s: %w", lastMigration.Version, err)
	}

	// Remove the migration record
	if _, err := tx.Exec("DELETE FROM schema_migrations WHERE version = $1", lastMigration.Version); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to remove migration record %s: %w", lastMigration.Version, err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit rollback %s: %w", lastMigration.Version, err)
	}

	log.Printf("Migration %s_%s rolled back successfully", lastMigration.Version, lastMigration.Name)
	return nil
}

// getAppliedMigrations returns a map of migration versions that have been applied
func getAppliedMigrations(db *sql.DB) (map[string]bool, error) {
	rows, err := db.Query("SELECT version FROM schema_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}

	return applied, rows.Err()
}
