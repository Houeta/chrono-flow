package bot

import (
	"log/slog"
	"testing"

	"github.com/Houeta/chrono-flow/test/mocks"
	"github.com/stretchr/testify/mock"
)

func TestStart(t *testing.T) {
	t.Parallel()

	mockBot := mocks.NewAPI(t)
	mockBot.On("Start").Once()

	logger := slog.Default()
	testBot := Bot{bot: mockBot, log: logger}

	testBot.Start()

	mockBot.AssertExpectations(t)
}

func TestStop(t *testing.T) {
	t.Parallel()

	mockBot := mocks.NewAPI(t)
	mockBot.On("Stop").Once()

	logger := slog.Default()
	testBot := Bot{bot: mockBot, log: logger}

	testBot.Stop()

	mockBot.AssertExpectations(t)
}

func TestRegisterRoutes(t *testing.T) {
	t.Parallel()

	mockBot := mocks.NewAPI(t)

	mockBot.On("Handle", "/start", mock.AnythingOfType("telebot.HandlerFunc")).Once()

	logger := slog.Default()
	testBot := Bot{bot: mockBot, log: logger}

	testBot.registerRoutes()

	mockBot.AssertExpectations(t)
}
