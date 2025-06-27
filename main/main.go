package main

import (
	"context"

	"github.com/vladyslavpavlenko/naholosy_bot/internal/bot"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/config"
	"github.com/vladyslavpavlenko/naholosy_bot/pkg/logger"
)

func main() {
	cfg := config.Must(config.NewFromEnv())
	l := logger.New(cfg.LogLevel)

	if err := bot.New(cfg, l).Run(context.Background()); err != nil {
		l.Fatal("bot stopped unexpectedly", logger.Error(err))
	}
}
