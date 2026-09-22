// Package app wires everything together.
package app

import (
	"context"
	"fmt"

	"github.com/mymmrac/telego"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/accent"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/bot"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/broadcast"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/config"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/handlers"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/metrics"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/practice"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/sender"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/stats"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/storage/sqlite"
	"github.com/vladyslavpavlenko/naholosy_bot/pkg/logger"
)

// Run starts the bot and blocks until the context is canceled.
func Run(ctx context.Context, cfg config.Config, l *logger.Logger) error {
	db, err := sqlite.Open(ctx, cfg.DBPath)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			l.Warn("closing database", logger.Error(closeErr))
		}
	}()

	words, err := sqlite.NewAccents(db).All(ctx)
	if err != nil {
		return err
	}

	catalog, err := accent.NewCatalog(words)
	if err != nil {
		return err
	}

	api, err := telego.NewBot(cfg.BotToken, telego.WithDiscardLogger())
	if err != nil {
		return fmt.Errorf("app: creating bot: %w", err)
	}

	me, err := api.GetMe(ctx)
	if err != nil {
		return fmt.Errorf("app: authorizing: %w", err)
	}

	l.Info("authorized",
		logger.Param("username", me.Username),
		logger.Param("words", catalog.Len()),
		logger.Param("database", cfg.DBPath),
	)

	m := metrics.New()
	users := sqlite.NewUsers(db)
	send := sender.New(api, m, l)

	return bot.New(cfg, users, handlers.New(
		users,
		practice.NewService(sqlite.NewSessions(db), sqlite.NewLearnedWords(db), catalog, m),
		stats.NewService(sqlite.NewStats(db), m, catalog),
		broadcast.NewService(users, send, m, l),
		catalog,
		send,
		m,
		l,
	), m, l).Run(ctx, api)
}
