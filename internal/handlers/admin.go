package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/vladyslavpavlenko/naholosy_bot/internal/broadcast"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/responses"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/sender"
	"github.com/vladyslavpavlenko/naholosy_bot/pkg/logger"
)

// Status reports on the bot.
func (h *Handlers) Status(ctx context.Context, req Request) error {
	report, err := h.stats.Report(ctx)
	if err != nil {
		return err
	}

	return h.sender.Send(ctx, sender.Message{ChatID: req.ChatID, Text: responses.Status(report)})
}

// BroadcastPrepare stages a broadcast and shows the admin what it will look
// like. Nothing is sent to anyone else until it is confirmed.
func (h *Handlers) BroadcastPrepare(ctx context.Context, req Request) error {
	payload, ok := h.payload(req)
	if !ok {
		return h.sender.Send(ctx, sender.Message{ChatID: req.ChatID, Text: responses.BroadcastUsage})
	}

	if err := h.preview(ctx, req.ChatID, payload); err != nil {
		return err
	}

	audience, err := h.broadcast.Audience(ctx)
	if err != nil {
		return err
	}

	h.broadcast.Prepare(req.User.ID, payload)

	return h.sender.Send(ctx, sender.Message{
		ChatID: req.ChatID,
		Text:   fmt.Sprintf(responses.BroadcastPrepared, len(audience)),
	})
}

// BroadcastTest sends the message to the admin alone.
func (h *Handlers) BroadcastTest(ctx context.Context, req Request) error {
	payload, ok := h.payload(req)
	if !ok {
		return h.sender.Send(ctx, sender.Message{ChatID: req.ChatID, Text: responses.BroadcastUsage})
	}

	if _, err := h.broadcast.Deliver(ctx, payload, []int64{req.User.ID}); err != nil {
		return err
	}

	return h.sender.Send(ctx, sender.Message{ChatID: req.ChatID, Text: responses.BroadcastTestSent})
}

// BroadcastConfirm sends the staged broadcast to everyone.
func (h *Handlers) BroadcastConfirm(ctx context.Context, req Request) error {
	payload, err := h.broadcast.Take(req.User.ID)
	if errors.Is(err, broadcast.ErrNothingPending) {
		return h.sender.Send(ctx, sender.Message{ChatID: req.ChatID, Text: responses.BroadcastNothing})
	}
	if err != nil {
		return err
	}

	audience, err := h.broadcast.Audience(ctx)
	if err != nil {
		return err
	}

	// Delivery outlives the update: a few hundred users paced under the rate
	// limit takes longer than an update should be held open for.
	h.background(func() {
		report, deliverErr := h.broadcast.Deliver(context.WithoutCancel(ctx), payload, audience)
		if deliverErr != nil {
			h.log.Error("broadcast interrupted", logger.Error(deliverErr))
		}

		if sendErr := h.sender.Send(context.WithoutCancel(ctx), sender.Message{
			ChatID: req.ChatID,
			Text: fmt.Sprintf(responses.BroadcastReport,
				report.Delivered, report.Failed, report.Total, report.Elapsed.Round(1e9)),
		}); sendErr != nil {
			h.log.Error("reporting broadcast failed", logger.Error(sendErr))
		}
	})

	return nil
}

// BroadcastCancel drops a staged broadcast.
func (h *Handlers) BroadcastCancel(ctx context.Context, req Request) error {
	_, err := h.broadcast.Take(req.User.ID)

	text := responses.BroadcastCanceled
	if errors.Is(err, broadcast.ErrNothingPending) {
		text = responses.BroadcastNothing
	}

	return h.sender.Send(ctx, sender.Message{ChatID: req.ChatID, Text: text})
}

// payload builds a broadcast from the command: the replied-to message if there
// is one, otherwise the text after the command.
func (h *Handlers) payload(req Request) (broadcast.Payload, bool) {
	if req.ReplyTo != nil {
		return broadcast.Payload{FromChatID: req.ReplyTo.ChatID, MessageID: req.ReplyTo.MessageID}, true
	}

	text := strings.TrimSpace(req.Args)
	if text == "" {
		return broadcast.Payload{}, false
	}

	return broadcast.Payload{Text: text}, true
}

// preview shows the admin exactly what recipients will get.
func (h *Handlers) preview(ctx context.Context, chatID int64, p broadcast.Payload) error {
	if p.IsCopy() {
		return h.sender.CopyMessage(ctx, chatID, p.FromChatID, p.MessageID)
	}

	return h.sender.Send(ctx, sender.Message{ChatID: chatID, Text: p.Text})
}
