package bot

import (
	"context"
	"fmt"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/config"
	"github.com/vladyslavpavlenko/naholosy_bot/pkg/logger"
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
		return err
	}

	updates, err := bot.UpdatesViaLongPolling(ctx, nil)
	if err != nil {
		return err
	}

	bh, err := th.NewBotHandler(bot, updates)
	if err != nil {
		return err
	}

	app.registerUpdates(bh)

	app.l.Info("bot started")
	if err = bh.Start(); err != nil {
		return err
	}

	<-ctx.Done()

	app.l.Info("bot stopped")
	return bh.Stop()
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
		app.l.Warn("unknown command", logger.Param("command", update.Message))
		_, _ = ctx.Bot().SendMessage(ctx, tu.Message(
			tu.ID(update.Message.Chat.ID),
			"Unknown command, use /start",
		))
		return nil
	}, th.AnyCommand())
}
