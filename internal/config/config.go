package config

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/viper"
)

var ErrEmptyToken = errors.New("error getting CF_TELEGRAM_TOKEN: variable not specified or contains an empty string")

type Config struct {
	Env         string // Env is the current environment: local, dev, prod.
	URL         string
	StoragePath string
	AllowedIDs  []int64
	Tg          Telegram
}

type Telegram struct {
	Token   string        // Token is an unique telgram bot token.
	Timeout time.Duration // Timeout is a poller timeout duration.
}

// MustLoad loads the configuration from environment variables and returns a Config struct.
func MustLoad() (*Config, error) {
	// Automatically binds environment variables to config keys
	viper.SetEnvPrefix("CF")
	viper.AutomaticEnv()

	// optional args
	viper.SetDefault("ENV", "production")
	viper.SetDefault("TELEGRAM_TIMEOUT", "15s")
	viper.SetDefault("STORAGE_PATH", "./chrono-flow.db")

	if viper.GetString("TELEGRAM_TOKEN") == "" {
		return nil, ErrEmptyToken
	}

	stringSlice := viper.GetStringSlice("ALLOWED_CHAT_IDS")
	allowedIDs, err := getInt64Slice(stringSlice)
	if err != nil {
		return nil, fmt.Errorf("failed to get allowed IDs from environment variables: %w", err)
	}

	return &Config{
		Env:         viper.GetString("ENV"),
		URL:         viper.GetString("DEST_URL"),
		StoragePath: viper.GetString("STORAGE_PATH"),
		AllowedIDs:  allowedIDs,
		Tg: Telegram{
			Token:   viper.GetString("TELEGRAM_TOKEN"),
			Timeout: viper.GetDuration("TELEGRAM_TIMEOUT"),
		},
	}, nil
}

func getInt64Slice(stringSlice []string) ([]int64, error) {
	int64Slice := make([]int64, 0, len(stringSlice))
	for _, s := range stringSlice {
		val, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing int64: %w", err)
		}
		int64Slice = append(int64Slice, val)
	}

	return int64Slice, nil
}
