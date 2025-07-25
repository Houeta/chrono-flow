package bot

import (
	"fmt"
	"log/slog"
	"time"

	"gopkg.in/telebot.v4"
)

// Bot contains the bot API instance and other information.
type Bot struct {
	bot *telebot.Bot
	log *slog.Logger
	// repo    repository.Interface
}

func NewBot(log *slog.Logger, token string, poller time.Duration) (*Bot, error) {
	bot, err := telebot.NewBot(telebot.Settings{
		Token:  token,
		Poller: &telebot.LongPoller{Timeout: poller},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Telegram bot: %w", err)
	}
	log.Info("Authorized on acount", "account", bot.Me.Username)

	botInstance := &Bot{bot: bot, log: log}

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
	b.bot.Handle("/start", b.startHandler)
}
