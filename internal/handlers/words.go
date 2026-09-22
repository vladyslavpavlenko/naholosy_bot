package handlers

import (
	"context"

	"github.com/vladyslavpavlenko/naholosy_bot/internal/keyboard"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/responses"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/sender"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/user"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/words"
)

// WordsMenu opens the word browser.
func (h *Handlers) WordsMenu(ctx context.Context, req Request) error {
	if err := h.moveTo(ctx, req.User, user.StageWords); err != nil {
		return err
	}

	return h.sender.Send(ctx, sender.Message{
		ChatID: req.ChatID,
		Text:   responses.WordsMenu,
		Markup: keyboard.Alphabet(h.catalog.Letters()),
	})
}

// SearchByLetters lists the words starting with the letters the user sent.
func (h *Handlers) SearchByLetters(ctx context.Context, req Request) error {
	letters := words.Letters(req.Text)

	return h.sender.Send(ctx, sender.Message{
		ChatID: req.ChatID,
		Text:   responses.WordList(letters, h.catalog.ByLetters(letters)),
		Markup: keyboard.Alphabet(h.catalog.Letters()),
	})
}

// Lookup answers a word the user typed with its stressed spelling.
func (h *Handlers) Lookup(ctx context.Context, req Request) error {
	found := h.catalog.Lookup(req.Text)

	text := responses.WordNotFound
	if len(found) > 0 {
		text = responses.Entries(found)
		h.metrics.LookupsHit.Add(1)
	} else {
		h.metrics.LookupsMiss.Add(1)
	}

	return h.sender.Send(ctx, sender.Message{
		ChatID:  req.ChatID,
		Text:    text,
		ReplyTo: req.MessageID,
	})
}
