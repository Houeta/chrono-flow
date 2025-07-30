package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Houeta/chrono-flow/internal/models"
	"github.com/Houeta/chrono-flow/internal/repository"
)

// GetState implements an interface method for retrieving state from the database.
func (r *Repository) GetState(ctx context.Context) (*models.State, error) {
	const opn = "repository.sqlite.GetState"

	// 1. Get hash of page
	var pageHash string
	err := r.db.QueryRowContext(ctx, "SELECT page_hash FROM page_state WHERE id = 1").Scan(&pageHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrStateNotFound
		}
		return nil, fmt.Errorf("%s: failed to get page hash: %w", opn, err)
	}

	// 2. Get all items from table
	rows, err := r.db.QueryContext(ctx, "SELECT model, type, quantity, price, image_url FROM products")
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get products: %w", opn, err)
	}
	defer rows.Close()

	// 3. Scan ecery row to Product structure
	var products []models.Product
	for rows.Next() {
		var p models.Product
		if err = rows.Scan(&p.Model, &p.Type, &p.Quantity, &p.Price, &p.ImageURL); err != nil {
			return nil, fmt.Errorf("%s: failed to scan product: %w", opn, err)
		}
		products = append(products, p)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", opn, err)
	}

	return &models.State{
		PageHash: pageHash,
		Products: products,
	}, nil
}

// UpdateState atomically updates the state using a transaction.
func (r *Repository) UpdateState(ctx context.Context, state *models.State) error {
	const opn = "storage.sqlite.UpdateState"

	// 1. begin transaction
	tx, err := r.db.BeginTx(ctx, nil) //nolint:varnamelen // tx its a default naming for transaction
	if err != nil {
		return fmt.Errorf("%s: failed to begin transaction: %w", opn, err)
	}
	defer tx.Rollback() //nolint:errcheck // Because in Go, it's common practice to ignore the Rollback() error in a defer, since if the transaction committed successfully, the rollback would just return sql.ErrTxDone and it's not useful to log or act on.

	// 2. Update (or insert) hash of page.
	_, err = tx.ExecContext(ctx, "INSERT OR REPLACE INTO page_state (id, page_hash) VALUES (1, ?)", state.PageHash)
	if err != nil {
		return fmt.Errorf("%s: failed to update page hash: %w", opn, err)
	}

	// 3. Completely clear the products table to record the new current state.
	_, err = tx.ExecContext(ctx, "DELETE FROM products")
	if err != nil {
		return fmt.Errorf("%s: failed to delete old products: %w", opn, err)
	}

	// 4. Preparing a request for the effective insertion of new products.
	stmt, err := tx.PrepareContext(
		ctx,
		"INSERT INTO products (model, type, quantity, price, image_url) VALUES (?, ?, ?, ?, ?)",
	)
	if err != nil {
		return fmt.Errorf("%s: failed to prepare insert statement: %w", opn, err)
	}
	defer stmt.Close()

	// 5. Insert each new product into the table.
	for _, p := range state.Products {
		if _, err = stmt.ExecContext(ctx, p.Model, p.Type, p.Quantity, p.Price, p.ImageURL); err != nil {
			return fmt.Errorf("%s: failed to insert product with model %s: %w", opn, p.Model, err)
		}
	}

	// 6. If all operations went through without errors - confirm the transaction.
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("%s: failed to commit transaction: %w", opn, err)
	}

	return nil
}
