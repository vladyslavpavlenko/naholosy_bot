package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/vladyslavpavlenko/naholosy_bot/internal/app"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/config"
	"github.com/vladyslavpavlenko/naholosy_bot/pkg/logger"
)

func main() {
	cfg, err := config.NewFromEnv()
	if err != nil {
		logger.New(logger.LevelProd).Fatal("invalid configuration", logger.Error(err))
	}

	l := logger.New(cfg.LogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err = app.Run(ctx, cfg, l); err != nil {
		l.Fatal("bot stopped unexpectedly", logger.Error(err))
	}
}
