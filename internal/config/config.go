package config

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Token    string  `envconfig:"TOKEN" required:"true"`
	LogLevel string  `envconfig:"LOG_LEVEL" default:"DEBUG"`
	AdminIDs []int64 `envconfig:"ADMIN_IDS"`
}

func Must(cfg Config, err error) Config {
	if err != nil {
		panic(err)
	}
	return cfg
}

func NewFromEnv() (cfg Config, err error) {
	if err := envconfig.Process("", &cfg); err != nil {
		return Config{}, fmt.Errorf("config parsing: %w", err)
	}
	return cfg, nil
}
