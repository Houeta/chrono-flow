package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Houeta/chrono-flow/internal/bot"
	"github.com/Houeta/chrono-flow/internal/config"
	"github.com/Houeta/chrono-flow/internal/parser"
	"github.com/Houeta/chrono-flow/internal/repository/sqlite"
	"github.com/Houeta/chrono-flow/internal/services/checker"
	_ "github.com/mattn/go-sqlite3"
)

// Constants for different environment types.
const (
	envLocal = "local"
	envDev   = "development"
	envProd  = "production"
)

// main is the entry point of the application.
func main() {
	// Create a context that will be canceled when an interrupt signal is received.
	// This allows for graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	// Load application configuration.
	cfg, err := config.MustLoad()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Set up the logger based on the environment.
	logger := setupLogger(ctx, cfg.Env)

	logger.InfoContext(ctx, "Initializing dependencies...")

	// Create a new parser
	parser := parser.NewParser(logger, cfg.URL)

	// Initialize the database connection.
	repo, err := sqlite.NewRepository(ctx, logger, cfg.StoragePath)
	if err != nil {
		logger.ErrorContext(ctx, "repository initialization failed", "error", err)
		os.Exit(1)
	}

	// Create a service which detects changes using repository and parser.
	updateChecker := checker.NewChecker(logger, parser, repo)

	// Create a telegram bot service
	notifier, err := bot.NewBot(logger, cfg.Tg.Token, cfg.Tg.Timeout, repo, cfg.AllowedIDs)
	if err != nil {
		logger.ErrorContext(ctx, "bot initialization failed", "error", err)
		os.Exit(1)
	}
	defer repo.Close()
	defer stop()

	// Log that the application has started.
	logger.InfoContext(
		ctx,
		"Starting main application loop. Press Ctrl+C to stop.",
		"interval",
		fmt.Sprintf("%dm", int(cfg.Interval.Minutes())),
	)

	// Start the bot's command handlers in a goroutine.
	go notifier.Start()
	defer notifier.Stop()

	// Run the first check immediately on startup without waiting for the first tick.
	runCheck(ctx, logger, updateChecker, notifier)

	// Run the main scheduler loop.
	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Triggered by the ticker for a scheduled check.
			runCheck(ctx, logger, updateChecker, notifier)

		case <-ctx.Done():
			// Triggered by Ctrl+C or another shutdown signal.
			logger.InfoContext(ctx, "Shutdown signal received. Stopping application...")
			return // Exit the loop and allow deferred functions to run.
		}
	}
}

// runCheck encapsulates the logic for a single update check.
func runCheck(ctx context.Context, log *slog.Logger, ch *checker.Checker, botNotifier *bot.Bot) {
	log.InfoContext(ctx, "Running scheduled check for updates...")

	// Perform the check.
	changes, err := ch.CheckForUpdates(ctx)
	if err != nil {
		log.ErrorContext(ctx, "failed to check for updates", "error", err)
		return
	}

	// If changes are found, send a notification.
	if changes.HasChanges() {
		log.InfoContext(ctx, "Changes detected, sending notification")
		if err = botNotifier.SendChangesNotification(ctx, changes); err != nil {
			log.ErrorContext(ctx, "failed to send notification", "error", err)
		}
	} else {
		log.InfoContext(ctx, "No new changes found")
	}
}

// setupLogger initializes and returns a logger based on the environment provided.
func setupLogger(ctx context.Context, env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level:     slog.LevelDebug,
				AddSource: true,
				ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
					return a
				},
			}),
		)
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level:     slog.LevelInfo,
				AddSource: false,
				ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
					return a
				},
			}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level:     slog.LevelWarn,
				AddSource: false,
				ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
					if a.Key == slog.TimeKey {
						return slog.Attr{}
					}
					return a
				},
			}),
		)
	default:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level:     slog.LevelError,
				AddSource: false,
				ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
					if a.Key == slog.TimeKey {
						return slog.Attr{}
					}
					return a
				},
			}),
		)

		log.ErrorContext(ctx,
			"The env parameter was not specified	 or was invalid. Logging will be minimal, by default.",
			slog.String("available_envs", "local, development, production"))
	}

	return log
}
