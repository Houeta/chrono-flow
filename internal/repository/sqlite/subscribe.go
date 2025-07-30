package sqlite

import (
	"context"
	"fmt"
)

// SubscribeChat adds the chat ID to the table.
func (r *Repository) SubscribeChat(ctx context.Context, chatID int64) error {
	const op = "repository.sqlite.SubcribeChat"
	_, err := r.db.ExecContext(ctx, "INSERT OR IGNORE INTO subscriptions (chat_id) VALUES (?)", chatID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// UnsubscribeChat deletes the chat ID from table.
func (r *Repository) UnsubscribeChat(ctx context.Context, chatID int64) error {
	const op = "repository.sqlite.UnsubscribeChat"
	_, err := r.db.ExecContext(ctx, "DELETE FROM subscriptions WHERE chat_id = ?", chatID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// GetSubscribedChats returns a slice of all subscribed chat IDs.
func (r *Repository) GetSubscribedChats(ctx context.Context) ([]int64, error) {
	const opn = "repository.sqlite.GetSubscribedChats"
	rows, err := r.db.QueryContext(ctx, "SELECT chat_id FROM subscriptions")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", opn, err)
	}
	defer rows.Close()

	var chatIDs []int64
	for rows.Next() {
		var id int64
		if err = rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("%s: failed to scan chat_id: %w", opn, err)
		}
		chatIDs = append(chatIDs, id)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", opn, err)
	}

	return chatIDs, nil
}
