package handlers

import (
	"context"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/keyboard"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/responses"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/sender"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/user"
)

// Start greets the user.
func (h *Handlers) Start(ctx context.Context, req Request) error {
	if err := h.moveTo(ctx, req.User, user.StageStart); err != nil {
		return err
	}

	return h.sender.Send(ctx, sender.Message{
		ChatID: req.ChatID,
		Text:   responses.Start,
		Markup: keyboard.Faster(),
	})
}

// MainMenu opens the main menu.
func (h *Handlers) MainMenu(ctx context.Context, req Request) error {
	if err := h.moveTo(ctx, req.User, user.StageMainMenu); err != nil {
		return err
	}

	return h.sender.Send(ctx, sender.Message{
		ChatID: req.ChatID,
		Text:   responses.MainMenu,
		Markup: keyboard.MainMenu(),
	})
}

// Download sends the handbook. The first send uploads it and remembers the
// file ID Telegram assigns, so later sends only reference it.
func (h *Handlers) Download(ctx context.Context, req Request) error {
	file := tu.FileFromBytes(handbook, "naholosy.pdf")
	if cached := h.handbookFileID.Load(); cached != nil {
		file = tu.FileFromID(*cached)
	}

	fileID, err := h.sender.SendDocument(ctx, req.ChatID, file)
	if err != nil {
		return err
	}

	if fileID != "" {
		h.handbookFileID.Store(&fileID)
	}

	return nil
}

// keyboardForStage returns the keyboard that belongs with a user's current
// stage, so that an error message does not strand them without one.
//
//nolint:ireturn // the markup differs per stage
func (h *Handlers) keyboardForStage(u *user.User) telego.ReplyMarkup {
	switch u.Stage {
	case user.StageMainMenu:
		return keyboard.MainMenu()
	case user.StageWords:
		return keyboard.Alphabet(h.catalog.Letters())
	case user.StagePracticeMenu:
		return keyboard.PracticeSizes()
	case user.StagePracticeReadiness:
		return keyboard.Readiness()
	case user.StageGame:
		return keyboard.FinishGame()
	case user.StageStart:
		return keyboard.Faster()
	default:
		return keyboard.Faster()
	}
}
