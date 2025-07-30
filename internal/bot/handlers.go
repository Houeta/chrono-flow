package bot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Houeta/chrono-flow/internal/models"
	"gopkg.in/telebot.v4"
)

const maxMessageLength = 4096

// subscribeHandler handles the /start or /subscribe command.
func (b *Bot) subscribeHandler(ctx telebot.Context) error {
	chatID := ctx.Chat().ID
	ctxRepo := context.Background()

	if !b.allowedChats[chatID] {
		b.log.Warn("Unathorized attempt to subscribe", "chatID", chatID)
		b.sendMessage(ctx, chatID, "👮 Sorry, this bot is private and cannot be used in this chat.")
		if err := b.bot.Leave(ctx.Recipient()); err != nil {
			return fmt.Errorf("failed to leave chat: %w", err)
		}

		return nil
	}

	if err := b.repo.SubscribeChat(ctxRepo, chatID); err != nil {
		b.log.Error("Failed to subscribe chat", "chatID", chatID, "err", err)
		b.sendMessage(ctx, chatID, "⛔ An internal error occurred. Failed to subscribe.")

		return nil
	}

	b.log.Info("Chat subscribed successfully", "chatID", chatID)
	b.sendMessage(ctx, chatID, "✅ You have successfully subscribed to updates!")

	return nil
}

// unsubscribeHandler handles the /start or /subscribe command.
func (b *Bot) unsubscribeHandler(ctx telebot.Context) error {
	chatID := ctx.Chat().ID
	repoCtx := context.Background()

	if err := b.repo.UnsubscribeChat(repoCtx, chatID); err != nil {
		b.log.Error("Failed to unsubscribe chat", "chatID", chatID)
		b.sendMessage(ctx, chatID, "⛔ An error occurred while trying to unsubscribe.")
		return fmt.Errorf("failed to unsubscribe chat: %w", err)
	}

	b.log.Info("Chat unsubscribed successfully", "chatID", chatID)
	b.sendMessage(ctx, chatID, "💔 You have unsubscribed from updates. To subscribe again, type /start or /subscribe.")
	return nil
}

// SendChangesNotification formats and sends the notification to all subscribers.
func (b *Bot) SendChangesNotification(ctx context.Context, changes *models.Changes) error {
	const opn = "bot.sendChangesNotification"
	const messageTimeout = 100
	log := b.log.With("op", opn)

	if !changes.HasChanges() {
		return nil
	}

	subscribers, err := b.repo.GetSubscribedChats(ctx)
	if err != nil {
		return fmt.Errorf("%s: failed to get subscribers: %w", opn, err)
	}

	if len(subscribers) == 0 {
		log.InfoContext(ctx, "No subscribers to notify")
		return nil
	}

	messageText := b.formatChangesMessage(changes)
	log.InfoContext(ctx, "Sending notification to subscribers", "count", len(subscribers))

	for _, chatID := range subscribers {
		recipient := &telebot.Chat{ID: chatID}
		_, err = b.bot.Send(recipient, messageText, telebot.ModeMarkdown)
		if err != nil {
			log.ErrorContext(ctx, "Failed to send notification to a chat", "chatID", chatID, "err", err)
		}
		time.Sleep(messageTimeout * time.Millisecond)
	}

	return nil
}

// formatChangesMessage builds the notification string from the changes.
func (b *Bot) formatChangesMessage(changes *models.Changes) string {
	var builder strings.Builder

	// Add a title with the current date.
	builder.WriteString(fmt.Sprintf("📅 *Product updates (%s)*\n\n", time.Now().Format("02.01.2006")))

	// Format added products.
	if len(changes.Added) > 0 {
		builder.WriteString(fmt.Sprintf("✅ *Added (%d):*\n", len(changes.Added)))
		for _, p := range changes.Added {
			builder.WriteString(
				fmt.Sprintf("• *Model*: `%s`\n  *Price*: %s, *Quantity*: %s\n", p.Model, p.Price, p.Quantity),
			)
		}
		builder.WriteString("\n")
	}

	// Format changed products.
	if len(changes.Changed) > 0 {
		builder.WriteString(fmt.Sprintf("🔄 *Changed (%d):*\n", len(changes.Changed)))
		for _, change := range changes.Changed {
			builder.WriteString(fmt.Sprintf("• *Model*: `%s`\n", change.New.Model))
			if change.New.Price != change.Old.Price {
				builder.WriteString(fmt.Sprintf("  *Price*: %s -> *%s*\n", change.Old.Price, change.New.Price))
			}
			if change.New.Quantity != change.Old.Quantity {
				builder.WriteString(fmt.Sprintf("  *Quantity*: %s -> *%s*\n", change.Old.Quantity, change.New.Quantity))
			}
			builder.WriteString("\n")
		}
		builder.WriteString("\n")
	}

	// Format removed products.
	if len(changes.Removed) > 0 {
		builder.WriteString(fmt.Sprintf("❌ *Removed (%d):*\n", len(changes.Removed)))
		for _, p := range changes.Removed {
			builder.WriteString(fmt.Sprintf("• *Model*: `%s`\n", p.Model))
		}
		builder.WriteString("\n")
	}

	// Truncate the message if it exceeds Telegram's limit.
	if builder.Len() > maxMessageLength {
		trimmedString := builder.String()[:maxMessageLength-50] // Leave space for the warning.
		return trimmedString + "\n\n... (the message was truncated)"
	}

	return builder.String()
}

// sendMessage - its a wrapper for sending a message.
func (b *Bot) sendMessage(ctx telebot.Context, chatID int64, text string) {
	err := ctx.Send(text)
	if err != nil {
		b.log.Error("Failed to send message", "chatID", chatID, "err", err)
	}
}
