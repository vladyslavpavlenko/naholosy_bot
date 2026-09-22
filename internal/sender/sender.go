// Package sender delivers messages to Telegram, retrying what is worth
// retrying and counting what is not.
package sender

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegoapi"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/metrics"
	"github.com/vladyslavpavlenko/naholosy_bot/pkg/logger"
)

// ErrBlocked means the user has blocked the bot or deleted the chat; there is
// no point retrying.
var ErrBlocked = errors.New("sender: chat unavailable")

const (
	// sendAttempts is how many times a send is tried before giving up.
	sendAttempts = 3
	// maxRetryAfter caps how long a 429 makes us wait before we give up
	// instead.
	maxRetryAfter = 30 * time.Second
)

// Sender sends messages.
type Sender struct {
	bot     *telego.Bot
	metrics *metrics.Metrics
	log     *logger.Logger
}

// New wires a sender.
func New(b *telego.Bot, m *metrics.Metrics, l *logger.Logger) *Sender {
	return &Sender{bot: b, metrics: m, log: l}
}

// Message is one outgoing message.
type Message struct {
	ChatID int64
	Text   string
	// Markup replaces the user's keyboard; nil leaves it alone.
	Markup telego.ReplyMarkup
	// ReplyTo quotes a message when non-zero.
	ReplyTo int
}

// Send delivers a text message, parsed as HTML.
func (s *Sender) Send(ctx context.Context, m Message) error {
	params := tu.Message(tu.ID(m.ChatID), m.Text).WithParseMode(telego.ModeHTML)

	if m.Markup != nil {
		params = params.WithReplyMarkup(m.Markup)
	}
	if m.ReplyTo != 0 {
		params = params.WithReplyParameters(&telego.ReplyParameters{
			MessageID:                m.ReplyTo,
			AllowSendingWithoutReply: true,
		})
	}

	return s.attempt(ctx, func() error {
		_, err := s.bot.SendMessage(ctx, params)
		return err
	})
}

// SetReaction reacts to a message with a single emoji.
//
// Reactions can be switched off per chat and are not worth a retry, so this
// returns the error for the caller to log rather than treating it as a failed
// send.
func (s *Sender) SetReaction(ctx context.Context, chatID int64, messageID int, emoji string) error {
	err := s.bot.SetMessageReaction(ctx, &telego.SetMessageReactionParams{
		ChatID:    tu.ID(chatID),
		MessageID: messageID,
		Reaction:  []telego.ReactionType{&telego.ReactionTypeEmoji{Type: telego.ReactionEmoji, Emoji: emoji}},
	})
	if err != nil {
		return fmt.Errorf("sender: reacting: %w", err)
	}

	return nil
}

// SendDocument delivers a document and returns the file ID Telegram assigned,
// so the caller can re-send it without uploading again.
func (s *Sender) SendDocument(ctx context.Context, chatID int64, file telego.InputFile) (string, error) {
	params := tu.Document(tu.ID(chatID), file)

	var fileID string
	err := s.attempt(ctx, func() error {
		msg, sendErr := s.bot.SendDocument(ctx, params)
		if sendErr == nil && msg != nil && msg.Document != nil {
			fileID = msg.Document.FileID
		}
		return sendErr
	})

	return fileID, err
}

// Quiz is a native quiz poll.
type Quiz struct {
	ChatID int64
	// Question is the prompt, up to 300 characters.
	Question string
	// Description sits under the question in smaller type, up to 1024
	// characters. It is where the run counter goes, so that the question
	// itself is nothing but the word.
	Description string
	// Options are the answers, 2 to 12 of them.
	Options []string
	// Correct is the 0-based index of the right answer. Telegram reveals it
	// on the client the moment the user picks, which is what makes a quiz
	// feel immediate.
	Correct int
	// Explanation is shown on a wrong answer, or when the lamp is tapped, up
	// to 200 characters.
	Explanation string
	// OpenPeriod is how long the poll accepts answers, 5 to 2628000 seconds.
	// The client counts down, and Telegram closes the poll when it runs out.
	OpenPeriod time.Duration
}

// SendQuiz sends a quiz poll and returns Telegram's ID for it, which is the
// only thing a poll answer comes back with.
//
// The poll is non-anonymous on purpose: Telegram sends no answer at all for an
// anonymous one, and answers are the whole point here.
func (s *Sender) SendQuiz(ctx context.Context, q Quiz) (string, error) {
	options := make([]telego.InputPollOption, 0, len(q.Options))
	for _, o := range q.Options {
		options = append(options, tu.PollOption(o))
	}

	anonymous := false

	params := tu.Poll(tu.ID(q.ChatID), q.Question, options...)
	params.Type = telego.PollTypeQuiz
	params.IsAnonymous = &anonymous
	params.CorrectOptionIDs = []int{q.Correct}
	params.Description = q.Description
	params.Explanation = q.Explanation
	params.ExplanationParseMode = telego.ModeHTML
	params.OpenPeriod = int(q.OpenPeriod.Seconds())

	var pollID string
	err := s.attempt(ctx, func() error {
		msg, sendErr := s.bot.SendPoll(ctx, params)
		if sendErr == nil && msg != nil && msg.Poll != nil {
			pollID = msg.Poll.ID
		}
		return sendErr
	})

	return pollID, err
}

// Edit replaces a message's text and drops its inline keyboard, which is how
// an answered question is marked up in place.
func (s *Sender) Edit(ctx context.Context, chatID int64, messageID int, text string) error {
	params := tu.EditMessageText(tu.ID(chatID), messageID, text).WithParseMode(telego.ModeHTML)

	return s.attempt(ctx, func() error {
		_, err := s.bot.EditMessageText(ctx, params)
		return err
	})
}

// AnswerCallback closes the loading state on the button the user tapped.
// Telegram spins it for a few seconds otherwise.
func (s *Sender) AnswerCallback(ctx context.Context, queryID string) error {
	if err := s.bot.AnswerCallbackQuery(ctx, tu.CallbackQuery(queryID)); err != nil {
		return fmt.Errorf("sender: answering callback: %w", err)
	}

	return nil
}

// SendText delivers a plain HTML message with no markup.
func (s *Sender) SendText(ctx context.Context, chatID int64, text string) error {
	return s.Send(ctx, Message{ChatID: chatID, Text: text})
}

// CopyMessage re-sends an existing message, keeping its formatting, media and
// markup. It is what lets a broadcast carry anything the admin can compose.
func (s *Sender) CopyMessage(ctx context.Context, chatID, fromChatID int64, messageID int) error {
	params := &telego.CopyMessageParams{
		ChatID:     tu.ID(chatID),
		FromChatID: tu.ID(fromChatID),
		MessageID:  messageID,
	}

	return s.attempt(ctx, func() error {
		_, err := s.bot.CopyMessage(ctx, params)
		return err
	})
}

// attempt runs a send, waiting out rate limits and giving up immediately on
// errors that will not get better.
func (s *Sender) attempt(ctx context.Context, send func() error) error {
	var err error

	for i := range sendAttempts {
		if err = send(); err == nil {
			s.metrics.MessagesSent.Add(1)
			return nil
		}

		if blocked(err) {
			s.metrics.SendFailures.Add(1)
			return fmt.Errorf("%w: %w", ErrBlocked, err)
		}

		wait, ok := retryAfter(err)
		if !ok || i == sendAttempts-1 {
			break
		}

		s.log.Warn("send rate limited, waiting",
			logger.Param("wait", wait.String()), logger.Param("attempt", i+1))

		select {
		case <-ctx.Done():
			s.metrics.SendFailures.Add(1)
			return ctx.Err()
		case <-time.After(wait):
		}
	}

	s.metrics.SendFailures.Add(1)

	return fmt.Errorf("sender: sending: %w", err)
}

// blocked reports whether Telegram says the chat is gone for good.
func blocked(err error) bool {
	var apiErr *telegoapi.Error
	if !errors.As(err, &apiErr) {
		return false
	}

	return apiErr.ErrorCode == 403
}

// retryAfter returns how long Telegram asked us to wait, if it did.
func retryAfter(err error) (time.Duration, bool) {
	var apiErr *telegoapi.Error
	if !errors.As(err, &apiErr) || apiErr.ErrorCode != 429 || apiErr.Parameters == nil {
		return 0, false
	}

	wait := time.Duration(apiErr.Parameters.RetryAfter) * time.Second
	if wait <= 0 || wait > maxRetryAfter {
		return 0, false
	}

	return wait, true
}
