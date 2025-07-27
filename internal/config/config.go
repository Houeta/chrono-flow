package config

import (
	"errors"
	"time"

	"github.com/spf13/viper"
)

var ErrEmptyToken = errors.New("error getting CF_TELEGRAM_TOKEN: variable not specified or contains an empty string")

type Config struct {
	Env string // Env is the current environment: local, dev, prod.
	URL string
	Tg  Telegram
}

type Telegram struct {
	Token   string        // Token is an unique telgram bot token.
	Timeout time.Duration // Timeout is a poller timeout duration.
}

// MustLoad loads the configuration from environment variables and returns a Config struct.
func MustLoad() *Config {
	// Automatically binds environment variables to config keys
	viper.SetEnvPrefix("CF")
	viper.AutomaticEnv()

	// optional args
	viper.SetDefault("ENV", "production")
	viper.SetDefault("TELEGRAM_TIMEOUT", "15s")

	if viper.GetString("TELEGRAM_TOKEN") == "" {
		panic(ErrEmptyToken)
	}

	return &Config{
		Env: viper.GetString("ENV"),
		URL: viper.GetString("DEST_URL"),
		Tg: Telegram{
			Token:   viper.GetString("TELEGRAM_TOKEN"),
			Timeout: viper.GetDuration("TELEGRAM_TIMEOUT"),
		},
	}
}
