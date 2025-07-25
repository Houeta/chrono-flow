package config_test

import (
	"testing"
	"time"

	"github.com/Houeta/chrono-flow/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestMustLoad(t *testing.T) {
	t.Run("error - empty required env variable", func(t *testing.T) {
		t.Setenv("CF_TELEGRAM_TOKEN", "")

		assert.PanicsWithError(t, config.ErrEmptyToken.Error(), func() {
			config.MustLoad()
		})
	})

	t.Run("success", func(t *testing.T) {
		t.Setenv("CF_ENV", "local")
		t.Setenv("CF_TELEGRAM_TOKEN", "telegramToken")

		cfg := config.MustLoad()

		assert.Equal(t, "local", cfg.Env)
		assert.Equal(t, 15*time.Second, cfg.Tg.Timeout)
		assert.Equal(t, "telegramToken", cfg.Tg.Token)
	})
}
