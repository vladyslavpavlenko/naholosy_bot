package main

import (
	"github.com/vladyslavpavlenko/naholosy_bot/internal/app/config"
	"github.com/vladyslavpavlenko/naholosy_bot/pkg/logger"
	"github.com/vladyslavpavlenko/naholosy_bot/pkg/logger/rotator"
)

func main() {
	cfg := config.Must(config.NewFromEnv())

	l := logger.NewWithRotation(cfg.LogLevel, &rotator.Options{
		MaxSize: 1,
	})

	l.Info("Started")
}
