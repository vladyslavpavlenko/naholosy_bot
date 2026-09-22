// Package bot is the Telegram runtime: it receives updates, works out which
// handler each one belongs to, and runs it.
package bot

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/config"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/handlers"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/handlers/predicates"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/metrics"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/user"
	"github.com/vladyslavpavlenko/naholosy_bot/pkg/logger"
)

const (
	// pollTimeout is how long Telegram holds a long poll open.
	pollTimeout = 30
	// handlerTimeout bounds one update. A handler sends a handful of messages
	// at most, so anything approaching this is stuck.
	handlerTimeout = 30 * time.Second
	// shutdownTimeout is how long in-flight updates get to finish.
	shutdownTimeout = 10 * time.Second
)

// Bot receives updates and dispatches them.
type Bot struct {
	cfg      config.Config
	users    user.Repository
	handlers *handlers.Handlers
	metrics  *metrics.Metrics
	log      *logger.Logger

	isAdmin th.Predicate
	locks   *KeyedMutex
	routes  map[string]handlers.Func
}

// New wires the bot runtime.
func New(
	cfg config.Config,
	users user.Repository,
	h *handlers.Handlers,
	m *metrics.Metrics,
	l *logger.Logger,
) *Bot {
	return &Bot{
		cfg:      cfg,
		users:    users,
		handlers: h,
		metrics:  m,
		log:      l,
		isAdmin:  predicates.Admin(&cfg),
		locks:    NewKeyedMutex(),
		routes: map[string]handlers.Func{
			RouteStart:            h.Start,
			RouteMenu:             h.MainMenu,
			RouteWordsMenu:        h.WordsMenu,
			RouteLetters:          h.SearchByLetters,
			RouteLookup:           h.Lookup,
			RouteDownload:         h.Download,
			RoutePracticeMenu:     h.PracticeMenu,
			RouteChooseSize:       h.ChooseSize,
			RouteStartRun:         h.StartRun,
			RouteAnswer:           h.Answer,
			RouteFinishRun:        h.FinishRun,
			RouteStatus:           h.Status,
			RouteBroadcast:        h.BroadcastPrepare,
			RouteBroadcastTest:    h.BroadcastTest,
			RouteBroadcastConfirm: h.BroadcastConfirm,
			RouteBroadcastCancel:  h.BroadcastCancel,
		},
	}
}

// Run receives updates until the context is canceled.
func (b *Bot) Run(ctx context.Context, api *telego.Bot) error {
	updates, err := api.UpdatesViaLongPolling(ctx, &telego.GetUpdatesParams{
		Timeout:        pollTimeout,
		AllowedUpdates: []string{"message", "poll_answer"},
	})
	if err != nil {
		return fmt.Errorf("bot: subscribing to updates: %w", err)
	}

	handler, err := th.NewBotHandler(api, updates)
	if err != nil {
		return fmt.Errorf("bot: creating handler: %w", err)
	}

	handler.Use(b.recover(), th.Timeout(handlerTimeout))
	handler.Handle(b.dispatchPollAnswer, th.AnyPollAnswer())
	handler.Handle(b.dispatch, th.AnyMessageWithText())

	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		<-ctx.Done()

		shutdown, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
		defer cancel()

		if stopErr := handler.StopWithContext(shutdown); stopErr != nil {
			b.log.Warn("stopping handler", logger.Error(stopErr))
		}
	}()

	b.log.Info("bot started")

	if err = handler.Start(); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("bot: handling updates: %w", err)
	}

	<-stopped
	b.log.Info("bot stopped")

	return nil
}

// recover keeps one bad update from taking the process down, which is what the
// original bot did on every unexpected database row.
func (b *Bot) recover() th.Handler {
	return th.PanicRecoveryHandler(func(recovered any) error {
		b.metrics.PanicsRecovered.Add(1)
		b.log.Error("recovered from panic", logger.Param("panic", fmt.Sprint(recovered)))

		return nil
	})
}

// dispatch resolves one message to a handler and runs it, holding the user's
// lock for the duration.
func (b *Bot) dispatch(ctx *th.Context, update telego.Update) error {
	msg := update.Message
	if msg == nil || msg.From == nil || msg.From.IsBot {
		return nil
	}

	b.metrics.UpdatesTotal.Add(1)

	release := b.locks.Lock(msg.From.ID)
	defer release()

	log := b.log.With(
		logger.Param("user_id", msg.From.ID),
		logger.Param("update_id", ctx.UpdateID()),
	)

	u, err := b.users.Ensure(ctx, msg.From.ID)
	if err != nil {
		b.metrics.UpdatesFailed.Add(1)
		log.Error("loading user failed", logger.Error(err))

		return nil
	}

	route := Resolve(u.Stage, msg.Text, b.isAdmin(ctx, update))
	if route == "" {
		log.Debug("message ignored", logger.Param("stage", u.Stage.String()))

		return nil
	}

	b.run(ctx, log, route, b.request(msg, u))

	return nil
}

// dispatchPollAnswer handles a pick in a quiz poll, which is how a practice
// answer arrives.
//
// A poll answer carries no chat, only the user. In a private chat those are
// the same number, and the bot only ever polls in private chats.
func (b *Bot) dispatchPollAnswer(ctx *th.Context, update telego.Update) error {
	answer := update.PollAnswer
	if answer.User == nil || len(answer.OptionIDs) == 0 {
		return nil
	}

	b.metrics.UpdatesTotal.Add(1)

	release := b.locks.Lock(answer.User.ID)
	defer release()

	log := b.log.With(
		logger.Param("user_id", answer.User.ID),
		logger.Param("update_id", ctx.UpdateID()),
	)

	u, err := b.users.Ensure(ctx, answer.User.ID)
	if err != nil {
		b.metrics.UpdatesFailed.Add(1)
		log.Error("loading user failed", logger.Error(err))

		return nil
	}

	if u.Stage != user.StageGame {
		log.Debug("poll answered outside a run", logger.Param("stage", u.Stage.String()))

		return nil
	}

	b.run(ctx, log, RouteAnswer, handlers.Request{
		ChatID: answer.User.ID,
		User:   u,
		Poll:   &handlers.PollAnswer{PollID: answer.PollID, Options: answer.OptionIDs},
	})

	return nil
}

// run executes a handler, timing it and reporting a failure to the user.
func (b *Bot) run(ctx *th.Context, log *logger.Logger, route string, req handlers.Request) {
	handle, ok := b.routes[route]
	if !ok {
		b.metrics.UpdatesFailed.Add(1)
		log.Error("no handler registered", logger.Param("route", route))

		return
	}

	started := time.Now()
	err := handle(ctx, req)
	b.metrics.ObserveHandler(route, time.Since(started), err != nil)

	if err == nil {
		log.Debug("handled", logger.Param("route", route))
		return
	}

	b.metrics.UpdatesFailed.Add(1)
	log.Error("handler failed", logger.Param("route", route), logger.Error(err))

	b.handlers.Fallback(ctx, req)
}

// request assembles what the handlers need from an update.
func (b *Bot) request(msg *telego.Message, u *user.User) handlers.Request {
	_, args := ParseCommand(msg.Text)

	req := handlers.Request{
		ChatID:    msg.Chat.ID,
		MessageID: msg.MessageID,
		Text:      msg.Text,
		Args:      args,
		User:      u,
	}

	if msg.ReplyToMessage != nil {
		req.ReplyTo = &handlers.Reply{
			ChatID:    msg.ReplyToMessage.Chat.ID,
			MessageID: msg.ReplyToMessage.MessageID,
		}
	}

	return req
}
