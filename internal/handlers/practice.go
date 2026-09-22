package handlers

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"

	"github.com/vladyslavpavlenko/naholosy_bot/internal/keyboard"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/practice"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/responses"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/sender"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/user"
	"github.com/vladyslavpavlenko/naholosy_bot/pkg/logger"
)

// PracticeMenu ends any run in progress and offers the run lengths.
func (h *Handlers) PracticeMenu(ctx context.Context, req Request) error {
	h.quizzes.forget(req.User.ID)

	if err := h.practice.Abandon(ctx, req.User.ID); err != nil {
		return err
	}

	if err := h.moveTo(ctx, req.User, user.StagePracticeMenu); err != nil {
		return err
	}

	return h.sender.Send(ctx, sender.Message{
		ChatID: req.ChatID,
		Text:   responses.PracticeMenu,
		Markup: keyboard.PracticeSizes(),
	})
}

// ChooseSize starts a run of the chosen length and explains the rules once.
func (h *Handlers) ChooseSize(ctx context.Context, req Request) error {
	size, err := strconv.Atoi(req.Text)
	if err != nil {
		return fmt.Errorf("handlers: unexpected run length %q: %w", req.Text, err)
	}

	if _, err = h.practice.Begin(ctx, req.User.ID, size); err != nil {
		return err
	}

	if moveErr := h.moveTo(ctx, req.User, user.StagePracticeReadiness); moveErr != nil {
		return moveErr
	}

	if req.User.TutorialSeen {
		return h.sender.Send(ctx, sender.Message{
			ChatID: req.ChatID,
			Text:   responses.PracticeExplanation3,
			Markup: keyboard.Readiness(),
		})
	}

	return h.tutorial(ctx, req, size)
}

// tutorial explains the rules, one message at a time.
func (h *Handlers) tutorial(ctx context.Context, req Request, size int) error {
	if err := h.sender.Send(ctx, sender.Message{
		ChatID: req.ChatID,
		Text:   fmt.Sprintf(responses.PracticeExplanation1, strconv.Itoa(size)),
		Markup: keyboard.Remove(),
	}); err != nil {
		return err
	}

	if err := h.sender.Send(ctx, sender.Message{
		ChatID: req.ChatID,
		Text:   responses.PracticeExplanation2,
	}); err != nil {
		return err
	}

	if err := h.sender.Send(ctx, sender.Message{
		ChatID: req.ChatID,
		Text:   responses.PracticeExplanation3,
		Markup: keyboard.Readiness(),
	}); err != nil {
		return err
	}

	req.User.SeeTutorial()

	return h.users.Save(ctx, req.User)
}

// StartRun puts up the standing keyboard and asks the first question.
//
// The keyboard needs a message of its own: the questions carry their variants
// as inline buttons, and one message can hold only one kind of markup.
func (h *Handlers) StartRun(ctx context.Context, req Request) error {
	if err := h.moveTo(ctx, req.User, user.StageGame); err != nil {
		return err
	}

	if err := h.sender.Send(ctx, sender.Message{
		ChatID: req.ChatID,
		Text:   responses.RunStarted,
		Markup: keyboard.FinishGame(),
	}); err != nil {
		return err
	}

	return h.ask(ctx, req)
}

// Answer grades a pick in a quiz poll and moves the run on. Telegram has
// already shown the user which option was right, so nothing is said here about
// the verdict itself.
func (h *Handlers) Answer(ctx context.Context, req Request) error {
	if req.Poll == nil || len(req.Poll.Options) == 0 {
		return nil
	}

	quiz, ok := h.quizzes.take(req.Poll.PollID)
	if !ok {
		// The poll was put up by an earlier process, so there is no way to
		// tell what the index means. Move the run on rather than leave the
		// user tapping at a question nothing answers.
		h.log.Debug("answer to an unknown poll", logger.Param("poll_id", req.Poll.PollID))

		return h.ask(ctx, req)
	}

	picked := req.Poll.Options[0]
	if picked < 0 || picked >= len(quiz.options) {
		return nil
	}

	result, err := h.practice.Answer(ctx, req.User.ID, quiz.options[picked])
	switch {
	case errors.Is(err, practice.ErrStaleAnswer):
		return nil
	case errors.Is(err, practice.ErrNoSession):
		// Marked as playing with nothing to play, which the old bot could
		// leave behind. Put the user back in the menu.
		return h.PracticeMenu(ctx, req)
	case err != nil:
		return err
	}

	if result.RunComplete {
		return h.complete(ctx, req)
	}

	return h.ask(ctx, req)
}

// FinishRun ends a run the user gave up on.
func (h *Handlers) FinishRun(ctx context.Context, req Request) error {
	summary, err := h.practice.Finish(ctx, req.User.ID)
	if errors.Is(err, practice.ErrNoSession) {
		return h.PracticeMenu(ctx, req)
	}
	if err != nil {
		return err
	}

	return h.results(ctx, req, summary)
}

// ask draws the next question and puts it as a quiz poll, which lets Telegram
// mark the right and wrong option on the client the moment the user picks.
func (h *Handlers) ask(ctx context.Context, req Request) error {
	question, err := h.practice.Question(ctx, req.User.ID)
	switch {
	case errors.Is(err, practice.ErrOutOfWords):
		if sendErr := h.sender.Send(ctx, sender.Message{
			ChatID: req.ChatID,
			Text:   responses.OutOfWords,
			Markup: keyboard.Remove(),
		}); sendErr != nil {
			return sendErr
		}

		return h.FinishRun(ctx, req)
	case errors.Is(err, practice.ErrNoSession):
		return h.PracticeMenu(ctx, req)
	case err != nil:
		return err
	}

	correct := slices.Index(question.Variants, question.Answer)
	if correct < 0 {
		return fmt.Errorf("handlers: %q is not among its own variants", question.Answer)
	}

	var note string
	if a, ok := h.catalog.ByAccented(question.Answer); ok {
		note = a.Note
	}

	pollID, err := h.sender.SendQuiz(ctx, sender.Quiz{
		ChatID:      req.ChatID,
		Question:    responses.QuizQuestion(question),
		Description: responses.QuizProgress(question),
		Options:     question.Variants,
		Correct:     correct,
		Explanation: responses.QuizExplanation(question.Answer, note),
	})
	if err != nil {
		return err
	}

	h.quizzes.put(pollID, req.User.ID, question.Variants)

	return nil
}

// complete reacts to a run that went the distance, then reports it.
func (h *Handlers) complete(ctx context.Context, req Request) error {
	summary, err := h.practice.Finish(ctx, req.User.ID)
	if err != nil {
		return err
	}

	if sendErr := h.sender.Send(ctx, sender.Message{
		ChatID: req.ChatID,
		Text:   responses.Reaction(summary.Grade),
		Markup: keyboard.Remove(),
	}); sendErr != nil {
		return sendErr
	}

	return h.results(ctx, req, summary)
}

// results reports the tally and returns the user to the practice menu.
func (h *Handlers) results(ctx context.Context, req Request, summary practice.Summary) error {
	if err := h.sender.Send(ctx, sender.Message{
		ChatID: req.ChatID,
		Text:   responses.Results(summary),
		Markup: keyboard.Remove(),
	}); err != nil {
		return err
	}

	return h.PracticeMenu(ctx, req)
}
