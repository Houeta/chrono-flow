package sqlite_test

import (
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Houeta/chrono-flow/internal/models"
	"github.com/Houeta/chrono-flow/internal/repository"
	"github.com/Houeta/chrono-flow/internal/repository/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Integration Tests (using a real temporary database)
// =============================================================================

// newTestDB is a helper function that creates a temporary database for a test.
func newTestDB(t *testing.T) sqlite.StateRepository {
	// t.Helper() marks this function as a test helper.
	t.Helper()

	// t.TempDir() creates a temporary directory that is automatically cleaned up after the test.
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Initialize the repository with the real, but temporary, database file.
	repo, err := sqlite.NewRepository(t.Context(), logger, dbPath)
	require.NoError(t, err, "failed to create test database")

	// t.Cleanup() registers a function to be called when the test finishes.
	t.Cleanup(func() {
		err = repo.Close()
		if err != nil {
			t.Logf("failed to close test database: %v", err)
		}
	})

	return repo
}

// TestRepository_Integration_UpdateAndGetState simulates the full lifecycle
// of the repository against a real SQLite database.
func TestRepository_Integration_UpdateAndGetState(t *testing.T) {
	// Arrange: Create a repository with a clean temporary database.
	repo := newTestDB(t)
	ctx := t.Context()

	// --- Scenario 1: Try to get state from an empty database ---
	t.Run("get_state_from_empty_db", func(t *testing.T) {
		// Act
		_, err := repo.GetState(ctx)
		// Assert: Expect the custom "not found" error.
		require.ErrorIs(t, err, repository.ErrStateNotFound)
	})

	// --- Scenario 2: Update state for the first time ---
	state1 := &models.State{
		PageHash: "hash1",
		Products: []models.Product{
			{Model: "A1", Price: "100"},
			{Model: "B2", Price: "200"},
		},
	}

	t.Run("update_state_first_time", func(t *testing.T) {
		// Act
		err := repo.UpdateState(ctx, state1)
		// Assert: Expect no error.
		require.NoError(t, err)
	})

	// --- Scenario 3: Get the saved state and verify it ---
	t.Run("get_state_after_first_update", func(t *testing.T) {
		// Act
		retrievedState, err := repo.GetState(ctx)
		// Assert
		require.NoError(t, err)
		require.NotNil(t, retrievedState)
		require.Equal(t, state1.PageHash, retrievedState.PageHash)
		// Use ElementsMatch for slices, as SQL does not guarantee order.
		require.ElementsMatch(t, state1.Products, retrievedState.Products)
	})

	// --- Scenario 4: Update state a second time (replacing all data) ---
	state2 := &models.State{
		PageHash: "hash2",
		Products: []models.Product{
			{Model: "C3", Price: "300"},
		},
	}

	t.Run("update_state_second_time", func(t *testing.T) {
		// Act
		err := repo.UpdateState(ctx, state2)
		// Assert
		require.NoError(t, err)
	})

	// --- Scenario 5: Get the second state and verify it ---
	t.Run("get_state_after_second_update", func(t *testing.T) {
		// Act
		retrievedState, err := repo.GetState(ctx)
		// Assert
		require.NoError(t, err)
		require.NotNil(t, retrievedState)
		require.Equal(t, state2.PageHash, retrievedState.PageHash)
		require.ElementsMatch(t, state2.Products, retrievedState.Products)
		require.Len(t, retrievedState.Products, 1) // Verify old products were deleted.
	})
}

// =============================================================================
// Unit Tests (using sqlmock for failure scenarios)
// =============================================================================

// newMockedRepo creates a repository with a mocked database connection for testing failures.
func newMockedRepo(t *testing.T) (*sqlite.Repository, sqlmock.Sqlmock) {
	t.Helper()

	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	// This assumes you have a constructor like `NewForTest` that accepts an existing *sql.DB.
	repo := sqlite.NewForTest(mockDB)

	t.Cleanup(func() { mockDB.Close() })

	return repo, mock
}

// TestRepository_GetState_Failures tests how GetState handles database errors.
func TestRepository_GetState_Failures(t *testing.T) {
	ctx := t.Context()

	t.Run("error_on_page_hash_query", func(t *testing.T) {
		// Arrange
		repo, mock := newMockedRepo(t)
		expectedErr := errors.New("db connection lost")
		// Expect a query for the page hash and return an error.
		mock.ExpectQuery("SELECT page_hash FROM page_state").WillReturnError(expectedErr)

		// Act
		_, err := repo.GetState(ctx)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), expectedErr.Error())
		assert.NoError(t, mock.ExpectationsWereMet()) // Verify all expectations were met.
	})

	t.Run("error_on_products_query", func(t *testing.T) {
		// Arrange
		repo, mock := newMockedRepo(t)
		// Expect a successful query for the page hash.
		hashRows := sqlmock.NewRows([]string{"page_hash"}).AddRow("test_hash")
		mock.ExpectQuery("SELECT page_hash FROM page_state").WillReturnRows(hashRows)

		// Expect a query for products and return an error.
		expectedErr := errors.New("table products is locked")
		mock.ExpectQuery("SELECT model, type, quantity, price, image_url FROM products").
			WillReturnError(expectedErr)

		// Act
		_, err := repo.GetState(ctx)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), expectedErr.Error())
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error_on_scan_query", func(t *testing.T) {
		// Arrange
		repo, mock := newMockedRepo(t)
		// Expect a successful query for the page hash.
		hashRows := sqlmock.NewRows([]string{"page_hash"}).AddRow("test_hash")
		mock.ExpectQuery("SELECT page_hash FROM page_state").WillReturnRows(hashRows)

		// Expect a query for products and return an error.
		productRows := sqlmock.NewRows([]string{"model", "type", "quantity", "price", "image_url"}).
			AddRow(nil, 123, 123, 123, 123)
		mock.ExpectQuery("SELECT model, type, quantity, price, image_url FROM products").WillReturnRows(productRows)

		// Act
		_, err := repo.GetState(ctx)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to scan product")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error_on_rows", func(t *testing.T) {
		// Arrange
		repo, mock := newMockedRepo(t)
		// Expect a successful query for the page hash.
		hashRows := sqlmock.NewRows([]string{"page_hash"}).AddRow("test_hash")
		mock.ExpectQuery("SELECT page_hash FROM page_state").WillReturnRows(hashRows)

		// Expect a query for products and return an error.
		productRows := sqlmock.NewRows([]string{"model", "type", "quantity", "price", "image_url"}).
			AddRow(123, 123, 123, 123, 123).
			RowError(0, assert.AnError)
		mock.ExpectQuery("SELECT model, type, quantity, price, image_url FROM products").WillReturnRows(productRows)

		// Act
		_, err := repo.GetState(ctx)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "rows iteration error")
		require.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestRepository_UpdateState_Failures tests how UpdateState handles transaction errors.
func TestRepository_UpdateState_Failures(t *testing.T) {
	ctx := t.Context()
	stateToUpdate := &models.State{
		PageHash: "new_hash",
		Products: []models.Product{{Model: "A1"}},
	}

	t.Run("error_on_begin_transaction", func(t *testing.T) {
		// Arrange
		repo, mock := newMockedRepo(t)
		expectedErr := errors.New("cannot start transaction")
		// Expect a call to Begin and return an error.
		mock.ExpectBegin().WillReturnError(expectedErr)

		// Act
		err := repo.UpdateState(ctx, stateToUpdate)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), expectedErr.Error())
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error_on_update_hash", func(t *testing.T) {
		// Arrange
		repo, mock := newMockedRepo(t)
		mock.ExpectBegin() // Expect successful Begin

		// Expect successful page_state update
		mock.ExpectExec("INSERT OR REPLACE INTO page_state").
			WithArgs(stateToUpdate.PageHash).
			WillReturnError(assert.AnError)

		// Because an error occurred, expect a Rollback.
		mock.ExpectRollback()

		// Act
		err := repo.UpdateState(ctx, stateToUpdate)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update page hash")
		require.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error_on_delete_products", func(t *testing.T) {
		// Arrange
		repo, mock := newMockedRepo(t)
		mock.ExpectBegin() // Expect successful Begin

		// Expect successful page_state update
		mock.ExpectExec("INSERT OR REPLACE INTO page_state").
			WithArgs(stateToUpdate.PageHash).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Expect the DELETE query and return an error.
		expectedErr := errors.New("delete failed")
		mock.ExpectExec("DELETE FROM products").
			WillReturnError(expectedErr)

		// Because an error occurred, expect a Rollback.
		mock.ExpectRollback()

		// Act
		err := repo.UpdateState(ctx, stateToUpdate)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete old products")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error_on_prepare_query", func(t *testing.T) {
		repo, mock := newMockedRepo(t)
		mock.ExpectBegin()
		mock.ExpectExec("INSERT OR REPLACE INTO page_state").WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec("DELETE FROM products").WillReturnResult(sqlmock.NewResult(0, 0))

		// Expect the method prepare returns an error
		mock.ExpectPrepare("INSERT INTO products").WillReturnError(assert.AnError)

		// Because an error occurred, expect a Rollback.
		mock.ExpectRollback()

		// Act
		err := repo.UpdateState(ctx, stateToUpdate)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to prepare insert statement")
		require.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error_on_insert_query", func(t *testing.T) {
		// Arrange
		repo, mock := newMockedRepo(t)
		mock.ExpectBegin()
		mock.ExpectExec("INSERT OR REPLACE INTO page_state").WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec("DELETE FROM products").WillReturnResult(sqlmock.NewResult(0, 0))

		// Expect the prepared statement and a successful execution.
		prep := mock.ExpectPrepare("INSERT INTO products")
		prep.ExpectExec().WithArgs("A1", "", "", "", "").WillReturnError(assert.AnError)

		// Because an error occurred, expect a Rollback.
		mock.ExpectRollback()

		// Act
		err := repo.UpdateState(ctx, stateToUpdate)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to insert product with model")
		require.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error_on_commit", func(t *testing.T) {
		// Arrange
		repo, mock := newMockedRepo(t)
		mock.ExpectBegin()
		mock.ExpectExec("INSERT OR REPLACE INTO page_state").WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec("DELETE FROM products").WillReturnResult(sqlmock.NewResult(0, 0))

		// Expect the prepared statement and a successful execution.
		prep := mock.ExpectPrepare("INSERT INTO products")
		prep.ExpectExec().WithArgs("A1", "", "", "", "").WillReturnResult(sqlmock.NewResult(1, 1))

		// Expect the final Commit call and return an error.
		expectedErr := errors.New("commit failed")
		mock.ExpectCommit().WillReturnError(expectedErr)

		// Act
		err := repo.UpdateState(ctx, stateToUpdate)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to commit transaction")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
