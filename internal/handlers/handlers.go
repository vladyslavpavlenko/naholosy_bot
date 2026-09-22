// Package handlers turns a routed update into the messages the bot sends back.
package handlers

import (
	"context"
	_ "embed"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/mymmrac/telego"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/accent"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/broadcast"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/metrics"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/practice"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/responses"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/sender"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/stats"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/user"
	"github.com/vladyslavpavlenko/naholosy_bot/pkg/logger"
)

//go:embed assets/naholosy.pdf
var handbook []byte

// timeoutBuffer is how many expired questions may wait to be dispatched. One
// per user in a run at once is the realistic ceiling; the buffer only has to
// keep a timer from blocking.
const timeoutBuffer = 64

// questionClock is how long a question lives. It is a field rather than a
// constant so that tests do not have to sit through it.
type questionClock struct {
	timeout time.Duration
	grace   time.Duration
}

// Request is one update, already resolved to the user it came from.
type Request struct {
	ChatID    int64
	MessageID int
	Text      string
	// Args is the text after a command, for the admin commands.
	Args string
	User *user.User
	// ReplyTo is the message the user replied to, used by /broadcast.
	ReplyTo *Reply
	// Poll is set when the update is an answer to a quiz poll rather than a
	// message.
	Poll *PollAnswer
}

// PollAnswer is a user's pick in a quiz poll.
type PollAnswer struct {
	// PollID is Telegram's identifier for the poll.
	PollID string
	// Options are the 0-based indexes picked. A quiz allows exactly one.
	Options []int
}

// Reply identifies a message an admin replied to.
type Reply struct {
	ChatID    int64
	MessageID int
}

// Func handles one request.
type Func func(ctx context.Context, req Request) error

// Sender delivers the bot's replies.
type Sender interface {
	Send(ctx context.Context, m sender.Message) error
	// SendQuiz sends a quiz poll and returns Telegram's ID for it.
	SendQuiz(ctx context.Context, q sender.Quiz) (string, error)
	SendDocument(ctx context.Context, chatID int64, file telego.InputFile) (string, error)
	CopyMessage(ctx context.Context, chatID, fromChatID int64, messageID int) error
}

// Handlers holds everything the handlers need.
type Handlers struct {
	users     user.Repository
	practice  *practice.Service
	stats     *stats.Service
	broadcast *broadcast.Service
	catalog   *accent.Catalog
	sender    Sender
	metrics   *metrics.Metrics
	log       *logger.Logger
	quizzes   *quizzes
	timeouts  chan Timeout
	clock     questionClock

	// handbookFileID caches the PDF's Telegram file ID after the first upload
	// so later sends do not re-upload it. It is shared by every user, which
	// the per-user lock does not cover, hence the atomic.
	handbookFileID atomic.Pointer[string]
}

// New wires the handlers.
func New(
	users user.Repository,
	practiceSvc *practice.Service,
	statsSvc *stats.Service,
	broadcastSvc *broadcast.Service,
	catalog *accent.Catalog,
	s Sender,
	m *metrics.Metrics,
	l *logger.Logger,
) *Handlers {
	return &Handlers{
		users:     users,
		practice:  practiceSvc,
		stats:     statsSvc,
		broadcast: broadcastSvc,
		catalog:   catalog,
		sender:    s,
		metrics:   m,
		log:       l,
		quizzes:   newQuizzes(),
		timeouts:  make(chan Timeout, timeoutBuffer),
		clock:     questionClock{timeout: questionTimeout, grace: timeoutGrace},
	}
}

// background runs work in its own goroutine, keeping a panic there from ending
// the process. The update handler's recovery covers only the goroutine it runs
// on, and this is not it.
func (h *Handlers) background(work func()) {
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				h.metrics.PanicsRecovered.Add(1)
				h.log.Error("recovered from panic in background work",
					logger.Param("panic", fmt.Sprint(recovered)))
			}
		}()

		work()
	}()
}

// Timeout announces a question that has run out of time.
type Timeout struct {
	UserID int64
	PollID string
}

// Timeouts is where questions that ran out of time are announced.
//
// Telegram closes a poll when its open period expires but sends no update
// about it — only a poll stopped by hand is reported — so the clock has to be
// ours. The bot reads this channel and dispatches each one the same way as an
// update, under the same per-user lock.
func (h *Handlers) Timeouts() <-chan Timeout {
	return h.timeouts
}

// Fallback tells the user something went wrong, without changing their state.
func (h *Handlers) Fallback(ctx context.Context, req Request) {
	if err := h.sender.Send(ctx, sender.Message{
		ChatID: req.ChatID,
		Text:   responses.SomethingWentWrong,
		Markup: h.keyboardForStage(req.User),
	}); err != nil {
		h.log.Error("sending fallback failed", logger.Error(err))
	}
}

// moveTo puts the user into a stage and persists it.
func (h *Handlers) moveTo(ctx context.Context, u *user.User, stage user.Stage) error {
	u.MoveTo(stage)

	return h.users.Save(ctx, u)
}
