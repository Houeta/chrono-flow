package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/Houeta/chrono-flow/internal/models"
)

// Repository represents a data repository that interacts with the database
// and provides logging capabilities. It holds a reference to the database
// and a logger instance for logging operations.
type Repository struct {
	db  *sql.DB
	log *slog.Logger
}

type StateRepository interface {
	// GetState returns the last saved state (page hash and product list).
	GetState(ctx context.Context) (*models.State, error)
	// UpdateState completely replaces the old state with the new one.
	UpdateState(ctx context.Context, state *models.State) error
}

type SubscribeRepository interface {
	// SubscribeChat adds a new chat to the list of subscribers.
	SubscribeChat(ctx context.Context, chatID int64) error

	// UnsubscribeChat removes a chat from the list of subscribers.
	UnsubscribeChat(ctx context.Context, chatID int64) error

	// GetSubscribedChats returns a list of all active subscribers.
	GetSubscribedChats(ctx context.Context) ([]int64, error)
}

// NewRepository creates a new instance of Repository with the provided Database.
// It returns a pointer to the newly created Repository.
func NewRepository(ctx context.Context, log *slog.Logger, storagePath string) (*Repository, error) {
	// Open (or create if it doesn't exist) the database file.
	dtb, err := sql.Open("sqlite3", fmt.Sprintf("%s?_pragma=foreign_keys(1)", storagePath))
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	// Check if the connection is actually established.
	if err = dtb.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("unable to establish connection to database: %w", err)
	}

	// Perform the initial schema migration.
	if err = initSchema(ctx, dtb); err != nil {
		return nil, fmt.Errorf("DB schema initialization error: %w", err)
	}

	return &Repository{db: dtb, log: log}, nil
}

// NewForTest creates a repository with an existing DB connection (for testing).
func NewForTest(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// initSchema creates the necessary tables if they don't already exist.
func initSchema(ctx context.Context, dtb *sql.DB) error {
	const migrationQuery = `
	CREATE TABLE IF NOT EXISTS page_state (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		page_hash TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS products (
		model TEXT PRIMARY KEY NOT NULL,
		type TEXT,
		quantity TEXT,
		price TEXT,
		image_url TEXT
	);

	CREATE TABLE IF NOT EXISTS subscriptions (
		chat_id INTEGER PRIMARY KEY NOT NULL,
		subscribed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err := dtb.ExecContext(ctx, migrationQuery)
	if err != nil {
		return fmt.Errorf("failed to execute migration query: %w", err)
	}

	return nil
}

// Close closes the connection to the database.
func (r *Repository) Close() error {
	if err := r.db.Close(); err != nil {
		r.log.Error("failed to close the database", "op", "repository.sqlite.Close", "error", err)
		return fmt.Errorf("failed to close the database: %w", err)
	}

	return nil
}

// DB is a getter for database handler.
func (r *Repository) DB() *sql.DB {
	return r.db
}
