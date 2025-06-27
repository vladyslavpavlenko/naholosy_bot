package bot

import (
	"context"
	"fmt"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/config"
	"github.com/vladyslavpavlenko/naholosy_bot/pkg/logger"
	"golang.org/x/sync/errgroup"
)

type App struct {
	cfg config.Config
	l   *logger.Logger
}

func New(cfg config.Config, l *logger.Logger) *App {
	return &App{cfg: cfg, l: l}
}

func (app *App) Run(ctx context.Context) error {
	app.l.Info("starting bot")
	defer app.l.Info("bot stopped")

	bot, err := telego.NewBot(app.cfg.Token)
	if err != nil {
		app.l.Error("new bot", logger.Error(err))
		return fmt.Errorf("new bot: %w", err)
	}

	updates, _ := bot.UpdatesViaLongPolling(ctx, nil)
	bh, _ := th.NewBotHandler(bot, updates)

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return bh.Start()
	})

	g.Go(func() error {
		app.l.Info("stopping bot...")
		<-ctx.Done()
		app.l.Info("bot stopped")
		return bh.Stop()
	})

	err = g.Wait()

	return err
}

func (app *App) registerUpdates(bh *th.BotHandler) {
	bh.Handle(func(ctx *th.Context, update telego.Update) error {
		_, _ = ctx.Bot().SendMessage(ctx, tu.Message(
			tu.ID(update.Message.Chat.ID),
			fmt.Sprintf("Hello %s!", update.Message.From.FirstName),
		))
		return nil
	}, th.CommandEqual("start"))

	bh.Handle(func(ctx *th.Context, update telego.Update) error {
		_, _ = ctx.Bot().SendMessage(ctx, tu.Message(
			tu.ID(update.Message.Chat.ID),
			"Unknown command, use /start",
		))
		return nil
	}, th.AnyCommand())
}
