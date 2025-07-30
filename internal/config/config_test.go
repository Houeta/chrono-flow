package config_test

import (
	"testing"
	"time"

	"github.com/Houeta/chrono-flow/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMustLoad(t *testing.T) {
	t.Run("error - empty required env variable", func(t *testing.T) {
		t.Setenv("CF_TELEGRAM_TOKEN", "")

		cfg, err := config.MustLoad()

		require.Error(t, err)
		assert.Nil(t, cfg)
		require.ErrorIs(t, err, config.ErrEmptyToken)
	})

	t.Run("error - empty required env variable", func(t *testing.T) {
		t.Setenv("CF_ALLOWED_CHAT_IDS", "invalid,id")
		t.Setenv("CF_TELEGRAM_TOKEN", "telegramToken")

		cfg, err := config.MustLoad()

		require.Error(t, err)
		assert.Nil(t, cfg)
		require.ErrorContains(t, err, "failed to get allowed ID")
		require.ErrorContains(t, err, "error parsing int64")
	})

	t.Run("success", func(t *testing.T) {
		t.Setenv("CF_ENV", "local")
		t.Setenv("CF_ALLOWED_CHAT_IDS", "-1234 -2345 -3456")
		t.Setenv("CF_TELEGRAM_TOKEN", "telegramToken")
		t.Setenv("CF_DEST_URL", "https://example.com")
		t.Setenv("CF_STORAGE_PATH", "some/path/to/db")

		cfg, err := config.MustLoad()

		require.NoError(t, err)
		assert.Equal(t, "local", cfg.Env)
		assert.Equal(t, 15*time.Second, cfg.Tg.Timeout)
		assert.Equal(t, "telegramToken", cfg.Tg.Token)
		assert.Equal(t, "https://example.com", cfg.URL)
		assert.Equal(t, "some/path/to/db", cfg.StoragePath)
		assert.Equal(t, []int64{-1234, -2345, -3456}, cfg.AllowedIDs)
	})
}
