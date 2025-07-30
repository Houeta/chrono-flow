package sqlite_test

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Unit Tests (using sqlmock for failure scenarios)
// =============================================================================

func TestSubscribeChat(t *testing.T) {
	ctx := t.Context()
	chatID := -123456789

	t.Run("error: exec query", func(t *testing.T) {
		// Arrange
		repo, mock := newMockedRepo(t)
		mock.ExpectExec("INSERT OR IGNORE INTO subscriptions").WillReturnError(assert.AnError)

		// Act
		err := repo.SubscribeChat(ctx, int64(chatID))

		// Assert
		require.Error(t, err)
		require.ErrorContains(t, err, "repository.sqlite.SubcribeChat")
		require.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success", func(t *testing.T) {
		// Arrange
		repo, mock := newMockedRepo(t)
		mock.ExpectExec("INSERT OR IGNORE INTO subscriptions").WillReturnResult(sqlmock.NewResult(1, 1))

		// Act
		err := repo.SubscribeChat(ctx, int64(chatID))

		// Assert
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUnsubscribeChat(t *testing.T) {
	ctx := t.Context()
	chatID := -123456789

	t.Run("error: exec query", func(t *testing.T) {
		// Arrange
		repo, mock := newMockedRepo(t)
		mock.ExpectExec("DELETE FROM subscriptions WHERE chat_id").WillReturnError(assert.AnError)

		// Act
		err := repo.UnsubscribeChat(ctx, int64(chatID))

		// Assert
		require.Error(t, err)
		require.ErrorContains(t, err, "repository.sqlite.UnsubscribeChat")
		require.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success", func(t *testing.T) {
		// Arrange
		repo, mock := newMockedRepo(t)
		mock.ExpectExec("DELETE FROM subscriptions WHERE chat_id").WillReturnResult(sqlmock.NewResult(1, 1))

		// Act
		err := repo.UnsubscribeChat(ctx, int64(chatID))

		// Assert
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGetSubscribedChats(t *testing.T) {
	ctx := t.Context()
	chatID := -123456789

	t.Run("error: cannot execute query", func(t *testing.T) {
		// Arrange
		repo, mock := newMockedRepo(t)
		mock.ExpectQuery("SELECT chat_id FROM subscriptions").WillReturnError(assert.AnError)

		// Act
		_, err := repo.GetSubscribedChats(ctx)

		// Assert
		require.Error(t, err)
		require.ErrorContains(t, err, "repository.sqlite.GetSubscribedChats")
		require.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error: failed to scan chat_id", func(t *testing.T) {
		// Arrange
		repo, mock := newMockedRepo(t)
		invalidRow := sqlmock.NewRows([]string{"chat_id"}).AddRow("invalid_id")
		mock.ExpectQuery("SELECT chat_id FROM subscriptions").WillReturnRows(invalidRow)

		// Act
		_, err := repo.GetSubscribedChats(ctx)

		// Assert
		require.Error(t, err)
		require.ErrorContains(t, err, "failed to scan chat_id")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error: rows error", func(t *testing.T) {
		// Arrange
		repo, mock := newMockedRepo(t)
		rowRithErr := sqlmock.NewRows([]string{"chat_id"}).AddRow(chatID).RowError(0, assert.AnError)
		mock.ExpectQuery("SELECT chat_id FROM subscriptions").WillReturnRows(rowRithErr)

		// Act
		_, err := repo.GetSubscribedChats(ctx)

		// Assert
		require.Error(t, err)
		require.ErrorContains(t, err, "rows iteration error")
		require.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("sucess", func(t *testing.T) {
		// Arrange
		repo, mock := newMockedRepo(t)
		validRow := sqlmock.NewRows([]string{"chat_id"}).AddRow(chatID)
		mock.ExpectQuery("SELECT chat_id FROM subscriptions").WillReturnRows(validRow)

		// Act
		chatIDs, err := repo.GetSubscribedChats(ctx)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, []int64{int64(chatID)}, chatIDs)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
