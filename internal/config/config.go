// Package config reads the bot's settings from the environment.
package config

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	BotToken string  `envconfig:"BOT_TOKEN" required:"true"`
	LogLevel string  `envconfig:"LOG_LEVEL" default:"DEBUG"`
	AdminIDs []int64 `envconfig:"ADMIN_IDS"`
	// DBPath is the SQLite file. It is the same file the first version of the
	// bot wrote, so replacing the binary is enough to keep every user's
	// progress.
	DBPath string `envconfig:"DB_PATH" default:"naholosy.db"`
}

func NewFromEnv() (cfg Config, err error) {
	if err := envconfig.Process("", &cfg); err != nil {
		return Config{}, fmt.Errorf("config parsing: %w", err)
	}

	return cfg, nil
}
