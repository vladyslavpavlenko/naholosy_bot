package practice

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"sync"

	"github.com/vladyslavpavlenko/naholosy_bot/internal/accent"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/metrics"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/words"
)

// ErrStaleAnswer is returned when a reply does not belong to the question
// currently open: a button tapped twice before the first tap was drawn, or one
// from a word already dealt with.
var ErrStaleAnswer = errors.New("practice: answer does not match the open question")

// ErrOutOfWords is returned when the word list cannot supply another word the
// run has not used. It cannot happen at the offered run lengths, but the run
// ends cleanly rather than looping if it ever does.
var ErrOutOfWords = errors.New("practice: no unasked words left")

const (
	// reviewPercent is how often, out of a hundred draws, the next word comes
	// from the words the user has already learned instead of the whole list.
	reviewPercent = 28

	// doublePercent is how often a question offers double-stress options at
	// all. Only five words in the list are genuinely double-stressed, so
	// without wrong ones mixed in a double-stress button would always be the
	// answer.
	doublePercent = 25

	// maxDoubles bounds how many are offered, to keep the keyboard short.
	maxDoubles = 2
)

// Service drives a practice run.
type Service struct {
	sessions Repository
	learned  LearnedWords
	catalog  *accent.Catalog
	metrics  *metrics.Metrics

	// missed holds the words each running session has got wrong, so that the
	// results can name them. The old schema has nowhere to put this, so it
	// lives here and is lost on restart: the results then simply omit it.
	mu     sync.Mutex
	missed map[int64][]string
}

// NewService wires a practice service.
func NewService(
	sessions Repository,
	learned LearnedWords,
	catalog *accent.Catalog,
	m *metrics.Metrics,
) *Service {
	return &Service{
		sessions: sessions,
		learned:  learned,
		catalog:  catalog,
		metrics:  m,
		missed:   make(map[int64][]string),
	}
}

// rememberMissed notes a word the user got wrong in their running session.
func (s *Service) rememberMissed(userID int64, word string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !slices.Contains(s.missed[userID], word) {
		s.missed[userID] = append(s.missed[userID], word)
	}
}

// takeMissed returns and clears the words the user's session got wrong.
func (s *Service) takeMissed(userID int64) []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	missed := s.missed[userID]
	delete(s.missed, userID)

	return missed
}

// Question is one prompt put to the user.
type Question struct {
	Number int
	Total  int
	// Word is the word to stress, spelled plainly.
	Word string
	// Answer is the correctly stressed spelling, always one of Variants. A
	// quiz has to be told which option is the right one.
	Answer string
	// Note is shown under the word, empty when there is none or when showing
	// it would give the answer away.
	Note string
	// Variants are the options to offer, correct answer included.
	Variants []string
}

// Result is the outcome of one reply.
type Result struct {
	Correct bool
	// Word is the correctly accented spelling of what was asked.
	Word        string
	RunComplete bool
	Session     *Session
}

// Summary is a finished run.
type Summary struct {
	Session *Session
	// Missed are the words the run got wrong at least once, in the order they
	// came up.
	Missed []string
	// Completed reports that the run went the distance rather than being
	// abandoned partway.
	Completed bool
	Grade     Grade
}

// Begin starts a run of the given length, discarding any run in progress.
func (s *Service) Begin(ctx context.Context, userID int64, size int) (*Session, error) {
	if err := s.sessions.Reset(ctx, userID); err != nil {
		return nil, err
	}

	s.takeMissed(userID)

	session, err := NewSession(userID, size)
	if err != nil {
		return nil, err
	}

	if saveErr := s.sessions.Save(ctx, session); saveErr != nil {
		return nil, saveErr
	}

	s.metrics.SessionsStarted.Add(1)

	return session, nil
}

// Question draws the next word and records it as asked.
func (s *Service) Question(ctx context.Context, userID int64) (Question, error) {
	session, err := s.sessions.Load(ctx, userID)
	if err != nil {
		return Question{}, err
	}

	drawn, err := s.draw(ctx, session)
	if err != nil {
		return Question{}, err
	}

	session.Ask(drawn.Word)
	if saveErr := s.sessions.Save(ctx, session); saveErr != nil {
		return Question{}, saveErr
	}

	variants, err := drawn.Variants(doubleOptions())
	if err != nil {
		return Question{}, fmt.Errorf("practice: variants for %q: %w", drawn.Word, err)
	}

	return Question{
		Number:   session.QuestionNumber(),
		Total:    session.Size,
		Word:     drawn.Plain(),
		Answer:   drawn.Word,
		Note:     questionNote(drawn),
		Variants: variants,
	}, nil
}

// doubleOptions decides how many double-stress options a question offers. The
// draw does not depend on the word, so the count says nothing about whether
// the answer is one of them.
func doubleOptions() int {
	if rand.IntN(100) >= doublePercent { //nolint:gosec // not a security decision
		return 0
	}

	return 1 + rand.IntN(maxDoubles) //nolint:gosec // not a security decision
}

// questionNote hides the note of a word that may be stressed more than one
// way, since "подвійний наголос" names the answer.
func questionNote(a accent.Accent) string {
	if a.DoubleStressed() {
		return ""
	}

	return a.Note
}

// draw picks the next word: usually from the whole list, sometimes from the
// words the user has learned. A word the run has already used is never drawn
// again.
func (s *Service) draw(ctx context.Context, session *Session) (accent.Accent, error) {
	asked := session.Asked()

	if rand.IntN(100) < reviewPercent { //nolint:gosec // not a security decision
		learned, err := s.learned.List(ctx, session.UserID)
		if err != nil {
			return accent.Accent{}, err
		}

		if drawn, ok := s.catalog.RandomAmong(learned, asked); ok {
			return drawn, nil
		}
	}

	drawn, ok := s.catalog.RandomExcept(asked)
	if !ok {
		return accent.Accent{}, ErrOutOfWords
	}

	return drawn, nil
}

// Answer records the user's reply to the question in progress.
func (s *Service) Answer(ctx context.Context, userID int64, given string) (Result, error) {
	session, err := s.sessions.Load(ctx, userID)
	if err != nil {
		return Result{}, err
	}

	word := session.CurrentWord
	if word == "" || words.Normalize(given) != words.Normalize(word) {
		return Result{}, ErrStaleAnswer
	}

	correct, ok := session.Answer(given)
	if !ok {
		return Result{}, ErrNoSession
	}

	if saveErr := s.sessions.Save(ctx, session); saveErr != nil {
		return Result{}, saveErr
	}

	if rememberErr := s.remember(ctx, userID, word, correct); rememberErr != nil {
		return Result{}, rememberErr
	}

	if correct {
		s.metrics.AnswersCorrect.Add(1)
	} else {
		s.metrics.AnswersWrong.Add(1)
		s.rememberMissed(userID, word)
	}

	return Result{Correct: correct, Word: word, RunComplete: session.Complete(), Session: session}, nil
}

// remember keeps the learned set in step with the latest answer.
func (s *Service) remember(ctx context.Context, userID int64, word string, correct bool) error {
	if correct {
		return s.learned.Add(ctx, userID, word)
	}

	return s.learned.Remove(ctx, userID, word)
}

// Finish closes the user's run and reports how it went. Only a run that went
// the distance is graded.
func (s *Service) Finish(ctx context.Context, userID int64) (Summary, error) {
	session, err := s.sessions.Load(ctx, userID)
	if err != nil {
		return Summary{}, err
	}

	summary := Summary{
		Session:   session,
		Missed:    s.takeMissed(userID),
		Completed: session.Complete(),
		Grade:     session.Grade(),
	}

	if summary.Completed {
		s.metrics.SessionsCompleted.Add(1)
	} else {
		s.metrics.SessionsAbandoned.Add(1)
	}

	return summary, nil
}

// Abandon drops any run in progress.
func (s *Service) Abandon(ctx context.Context, userID int64) error {
	s.takeMissed(userID)

	return s.sessions.Reset(ctx, userID)
}
