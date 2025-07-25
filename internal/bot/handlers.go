package bot

import (
	"fmt"

	"gopkg.in/telebot.v4"
)

// startHandler process command /start.
func (b *Bot) startHandler(ctx telebot.Context) error {
	b.log.Info("User started the bot", "username", ctx.Sender().Username)

	err := ctx.Send("Hello!")
	if err != nil {
		return fmt.Errorf("failed to send greeting message: %w", err)
	}

	return nil
}
