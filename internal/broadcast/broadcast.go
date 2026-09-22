// Package broadcast sends one message to every user of the bot.
package broadcast

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/vladyslavpavlenko/naholosy_bot/internal/metrics"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/user"
	"github.com/vladyslavpavlenko/naholosy_bot/pkg/logger"
)

// ErrNothingPending is returned when a broadcast is confirmed without one
// having been prepared.
var ErrNothingPending = errors.New("broadcast: nothing prepared")

// pace is the gap between deliveries, keeping well under Telegram's limit of
// about 30 messages a second to different chats.
const pace = 40 * time.Millisecond

// Payload is what gets sent. A payload either copies an existing message,
// which preserves its formatting, media and inline keyboard, or carries HTML
// text of its own.
type Payload struct {
	Text       string
	FromChatID int64
	MessageID  int
}

// IsCopy reports whether the payload re-sends an existing message.
func (p Payload) IsCopy() bool { return p.MessageID != 0 }

// Sender delivers one payload to one user.
type Sender interface {
	SendText(ctx context.Context, userID int64, text string) error
	CopyMessage(ctx context.Context, userID, fromChatID int64, messageID int) error
}

// Report is the outcome of a broadcast.
type Report struct {
	Total     int
	Delivered int
	Failed    int
	Elapsed   time.Duration
}

// Service prepares and delivers broadcasts. A prepared broadcast is held in
// memory only, so a restart cancels it rather than sending it by surprise.
type Service struct {
	users   user.Repository
	sender  Sender
	metrics *metrics.Metrics
	log     *logger.Logger

	mu      sync.Mutex
	pending map[int64]Payload
}

// NewService wires a broadcast service.
func NewService(users user.Repository, sender Sender, m *metrics.Metrics, l *logger.Logger) *Service {
	return &Service{users: users, sender: sender, metrics: m, log: l, pending: make(map[int64]Payload)}
}

// Prepare stages a payload for an admin to confirm.
func (s *Service) Prepare(adminID int64, p Payload) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.pending[adminID] = p
}

// Take returns and clears the payload an admin staged.
func (s *Service) Take(adminID int64) (Payload, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.pending[adminID]
	if !ok {
		return Payload{}, ErrNothingPending
	}
	delete(s.pending, adminID)

	return p, nil
}

// Audience returns everyone a broadcast would reach.
func (s *Service) Audience(ctx context.Context) ([]int64, error) {
	return s.users.IDs(ctx)
}

// Deliver sends the payload to each target in turn, paced to stay under
// Telegram's rate limit. A failure for one user is logged and counted; the
// rest still get the message.
func (s *Service) Deliver(ctx context.Context, p Payload, targets []int64) (Report, error) {
	report := Report{Total: len(targets)}
	started := time.Now()

	ticker := time.NewTicker(pace)
	defer ticker.Stop()

	for i, id := range targets {
		if i > 0 {
			select {
			case <-ctx.Done():
				report.Elapsed = time.Since(started)
				return report, ctx.Err()
			case <-ticker.C:
			}
		}

		if err := s.deliverOne(ctx, p, id); err != nil {
			report.Failed++
			s.metrics.BroadcastsFailed.Add(1)
			s.log.Warn("broadcast delivery failed", logger.Param("user_id", id), logger.Error(err))
			continue
		}

		report.Delivered++
		s.metrics.BroadcastsDelivered.Add(1)
	}

	report.Elapsed = time.Since(started)

	return report, nil
}

func (s *Service) deliverOne(ctx context.Context, p Payload, userID int64) error {
	if p.IsCopy() {
		return s.sender.CopyMessage(ctx, userID, p.FromChatID, p.MessageID)
	}

	if p.Text == "" {
		return fmt.Errorf("broadcast: empty payload")
	}

	return s.sender.SendText(ctx, userID, p.Text)
}
