package sqlite_test

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/Houeta/chrono-flow/internal/repository/sqlite"
	_ "github.com/mattn/go-sqlite3"
)

func TestNewRepository_Success(t *testing.T) {
	ctx := t.Context()

	// Create a temporary file to act as the SQLite DB
	tmpFile, err := os.CreateTemp(t.TempDir(), "testdb-*.sqlite")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name()) // clean up after test

	// No-op logger
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	repo, err := sqlite.NewRepository(ctx, logger, tmpFile.Name())
	if err != nil {
		t.Fatalf("expected no error from NewRepository, got: %v", err)
	}
	defer repo.Close()

	// Check that repository is not nil
	if repo == nil {
		t.Fatal("expected repository to be non-nil")
	}
}

func TestNewRepository_InvalidPath(t *testing.T) {
	ctx := t.Context()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Try creating a repo with an invalid file path
	_, err := sqlite.NewRepository(ctx, logger, "/invalid/path/to/db.sqlite")
	if err == nil {
		t.Fatal("expected error due to invalid path, got nil")
	}
}

func TestRepository_Close(t *testing.T) {
	ctx := t.Context()

	tmpFile, err := os.CreateTemp(t.TempDir(), "testdb-*.sqlite")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	repo, err := sqlite.NewRepository(ctx, logger, tmpFile.Name())
	if err != nil {
		t.Fatalf("expected no error from NewRepository, got: %v", err)
	}

	if err = repo.Close(); err != nil {
		t.Fatalf("expected no error on Close, got: %v", err)
	}
}

func TestSchemaInitialization(t *testing.T) {
	ctx := t.Context()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "schema-test.sqlite")

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	repo, err := sqlite.NewRepository(ctx, logger, dbPath)
	if err != nil {
		t.Fatalf("expected no error from NewRepository, got: %v", err)
	}
	defer repo.Close()

	rows, err := repo.DB().Query("SELECT name FROM sqlite_master WHERE type='table'")
	if err != nil {
		t.Fatalf("failed to query sqlite_master: %v", err)
	}
	defer rows.Close()

	found := make(map[string]bool)
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			t.Fatalf("failed to scan row: %v", err)
		}
		found[name] = true
	}

	if !found["page_state"] || !found["products"] {
		t.Errorf("expected tables 'page_state' and 'products' to exist, got: %+v", found)
	}
}
