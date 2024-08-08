package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

const operation = "config parsing"

// Config is the application configuration structure.
type Config struct {
	Token    string `env:"TOKEN,required,notEmpty"`
	LogLevel string `env:"LOG_LEVEL,required,notEmpty"`
}

// Must is a wrapper around return results from the NewFromEnv()
// function, which will panic in case of error.
func Must(cfg *Config, err error) *Config {
	if err != nil {
		panic(err)
	}
	return cfg
}

// NewFromEnv parses the environment variables into the Config struct.
// Returns an error if any of the required variables is missing or contains
// an invalid value.
func NewFromEnv() (*Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("%s failed: %w", operation, err)
	}
	return &cfg, nil
}
