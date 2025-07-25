package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Houeta/chrono-flow/internal/bot"
	"github.com/Houeta/chrono-flow/internal/config"
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

	cfg := config.MustLoad()

	// Set up the logger based on the environment.
	logger := setupLogger(cfg.Env)

	chronoBot, err := bot.NewBot(logger, cfg.Tg.Token, cfg.Tg.Timeout)
	if err != nil {
		log.Fatalf("Failed to init bot: %v", err)
	}
	defer stop()

	// Log that the application has started.
	logger.InfoContext(ctx, "Application started. Press Ctrl+C to stop.")

	// Start the bot in a goroutine to allow main to listen for signals.
	go chronoBot.Start()

	// Wait for the context to be canceled (e.g., by Ctrl+C).
	<-ctx.Done()

	// Log that a shutdown signal has been received.
	logger.InfoContext(ctx, "Shutdown signal received. Stopping application...")

	// Stop the bot gracefully.
	chronoBot.Stop()

	// Log graceful shutdown completion.
	logger.InfoContext(ctx, "Application stopped gracefully.")
}

// setupLogger initializes and returns a logger based on the environment provided.
func setupLogger(env string) *slog.Logger {
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

		log.Error(
			"The env parameter was not specified	 or was invalid. Logging will be minimal, by default.",
			slog.String("available_envs", "local, development, production"))
	}

	return log
}
