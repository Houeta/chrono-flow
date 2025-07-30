package checker

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/Houeta/chrono-flow/internal/models"
	"github.com/Houeta/chrono-flow/internal/parser"
	"github.com/Houeta/chrono-flow/internal/repository"
	"github.com/Houeta/chrono-flow/internal/repository/sqlite"
)

// Checker is an orchestrator that performs a full verification cycle.
type Checker struct {
	log    *slog.Logger
	parser parser.HTMLParser
	repo   sqlite.StateRepository
}

type Interface interface {
	// CheckForUpdates performs the full change checking algorithm.
	CheckForUpdates(ctx context.Context) (*models.Changes, error)
}

// NewChecker creates a new Checker instance.
func NewChecker(log *slog.Logger, parser parser.HTMLParser, repo sqlite.StateRepository) *Checker {
	return &Checker{log: log, parser: parser, repo: repo}
}

// CheckForUpdates performs the full change checking algorithm.
func (c *Checker) CheckForUpdates(ctx context.Context) (*models.Changes, error) {
	const opn = "checker.CheckForUpdates"
	log := c.log.With("op", opn)

	// 1. Retrieving HTML and calculating a new hash
	log.InfoContext(ctx, "Fetching HTML page to check for updates")
	resp, err := c.parser.GetHTMLResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get html response: %w", opn, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to read response body: %w", opn, err)
	}

	newPageHash := calculateHash(body)
	log.DebugContext(ctx, "Calculated new page hash", "hash", newPageHash)

	// 2. Getting the old state from the database
	oldState, err := c.repo.GetState(ctx)
	if err != nil && !errors.Is(err, repository.ErrStateNotFound) {
		return nil, fmt.Errorf("%s: failed to get old state: %w", opn, err)
	}

	// 3. Hash comparison
	if err == nil && oldState.PageHash == newPageHash {
		log.InfoContext(ctx, "Page hash has not changed. No updates.")
		return &models.Changes{}, nil
	}
	log.InfoContext(ctx, "Page hash differs or first run. Starting full analysis...")

	// 4. Full page parsing
	newProducts, err := c.parser.ParseTableResponse(ctx, io.NopCloser(bytes.NewReader(body)))
	if err != nil {
		return nil, fmt.Errorf("%s: failed to parse products from new response: %w", opn, err)
	}
	log.InfoContext(ctx, "Successfully parsed products", "count", len(newProducts))

	// 5. Product list comparison
	var oldProducts []models.Product
	if oldState != nil {
		oldProducts = oldState.Products
	}
	changes := detectChanges(oldProducts, newProducts)
	log.InfoContext(
		ctx,
		"Change detection complete",
		"added",
		len(changes.Added),
		"removed",
		len(changes.Removed),
		"changed",
		len(changes.Changed),
	)

	// 6. Updating the database and returning the result
	newState := &models.State{
		PageHash: newPageHash,
		Products: newProducts,
	}

	if err = c.repo.UpdateState(ctx, newState); err != nil {
		return nil, fmt.Errorf("%s: failed to update state in repository: %w", opn, err)
	}
	log.InfoContext(ctx, "Successfully updated state in repository")

	return &changes, nil
}

// calculateHash calculates the SHA256 hash for a slice of bytes.
func calculateHash(data []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

// detectChanges compares two product lists and finds the difference.
func detectChanges(oldProducts, newProducts []models.Product) models.Changes {
	oldMap := make(map[string]models.Product, len(oldProducts))
	for _, p := range oldProducts {
		oldMap[p.Model] = p
	}

	newMap := make(map[string]models.Product, len(newProducts))
	for _, p := range newProducts {
		newMap[p.Model] = p
	}

	var changes models.Changes
	for newModel, newProduct := range newMap {
		oldProduct, found := oldMap[newModel]
		if found {
			if newProduct.Price != oldProduct.Price || newProduct.Quantity != oldProduct.Quantity {
				changes.Changed = append(changes.Changed, models.ChangeInfo{Old: oldProduct, New: newProduct})
			}
			delete(oldMap, newModel)
		} else {
			changes.Added = append(changes.Added, newProduct)
		}
	}

	for _, removedProduct := range oldMap {
		changes.Removed = append(changes.Removed, removedProduct)
	}
	return changes
}
