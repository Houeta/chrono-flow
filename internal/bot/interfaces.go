package bot

import "gopkg.in/telebot.v4"

type API interface {
	// Handle lets you set the handler for some command name or one of the supported endpoints. It also applies middleware if such passed to the function.
	Handle(endpoint interface{}, h telebot.HandlerFunc, m ...telebot.MiddlewareFunc)
	// Start brings bot into motion by consuming incoming updates (see Bot.Updates channel).
	Start()
	// Stop gracefully shuts the poller down.
	Stop()

	Leave(chat telebot.Recipient) error

	NewContext(u telebot.Update) telebot.Context

	Send(to telebot.Recipient, what interface{}, opts ...interface{}) (*telebot.Message, error)
}
