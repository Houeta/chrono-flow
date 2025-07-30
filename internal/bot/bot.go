package bot

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/Houeta/chrono-flow/internal/repository/sqlite"
	"gopkg.in/telebot.v4"
)

// Bot contains the bot API instance and other information.
type Bot struct {
	bot          API
	log          *slog.Logger
	repo         sqlite.SubscribeRepository
	allowedChats map[int64]bool
}

func NewBot(
	log *slog.Logger,
	token string,
	poller time.Duration,
	repo sqlite.SubscribeRepository,
	allowedIDs []int64,
) (*Bot, error) {
	bot, err := telebot.NewBot(telebot.Settings{
		Token:  token,
		Poller: &telebot.LongPoller{Timeout: poller},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Telegram bot: %w", err)
	}
	log.Info("Authorized on account", "account", bot.Me.Username)

	allowedMap := make(map[int64]bool)
	for _, id := range allowedIDs {
		allowedMap[id] = true
	}

	botInstance := &Bot{bot: bot, log: log, allowedChats: allowedMap, repo: repo}
	botInstance.registerRoutes()

	return botInstance, nil
}

// Start launches the bot to listen for updates.
func (b *Bot) Start() {
	b.log.Info("Telegram bot is starting...")
	b.bot.Start()
}

// Stop gracefully stops the Telegram bot and logs the action.
func (b *Bot) Stop() {
	b.log.Info("Telegram bot is stopped...")
	b.bot.Stop()
}

// registerRoutes configures all routes (commands).
func (b *Bot) registerRoutes() {
	// Public routes.
	b.bot.Handle("/start", b.subscribeHandler)
	b.bot.Handle("/subscribe", b.subscribeHandler)
	b.bot.Handle("/unsubscribe", b.unsubscribeHandler)
}
