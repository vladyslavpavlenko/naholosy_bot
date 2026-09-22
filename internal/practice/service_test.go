package practice_test

import (
	"context"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/accent"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/metrics"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/practice"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/user"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/words"
)

// fakeUsers is an in-memory [user.Repository].
type fakeUsers struct {
	users map[int64]*user.User
}

func (f *fakeUsers) Ensure(_ context.Context, id int64) (*user.User, error) {
	if u, ok := f.users[id]; ok {
		return u, nil
	}

	u := &user.User{ID: id, Stage: user.StageStart}
	f.users[id] = u

	return u, nil
}

func (f *fakeUsers) Get(_ context.Context, id int64) (*user.User, error) {
	u, ok := f.users[id]
	if !ok {
		return nil, user.ErrNotFound
	}

	return u, nil
}

func (f *fakeUsers) Save(_ context.Context, u *user.User) error {
	copied := *u
	f.users[u.ID] = &copied

	return nil
}

func (f *fakeUsers) IDs(context.Context) ([]int64, error) {
	ids := make([]int64, 0, len(f.users))
	for id := range f.users {
		ids = append(ids, id)
	}
	slices.Sort(ids)

	return ids, nil
}

// fakeSessions is an in-memory [practice.Repository].
type fakeSessions struct {
	sessions map[int64]*practice.Session
}

func (f *fakeSessions) Load(_ context.Context, userID int64) (*practice.Session, error) {
	s, ok := f.sessions[userID]
	if !ok {
		return nil, practice.ErrNoSession
	}

	return s, nil
}

func (f *fakeSessions) Save(_ context.Context, s *practice.Session) error {
	f.sessions[s.UserID] = s

	return nil
}

func (f *fakeSessions) Reset(_ context.Context, userID int64) error {
	delete(f.sessions, userID)

	return nil
}

// fakeLearned is an in-memory [practice.LearnedWords].
type fakeLearned struct {
	words map[int64][]string
}

func (f *fakeLearned) List(_ context.Context, userID int64) ([]string, error) {
	return f.words[userID], nil
}

func (f *fakeLearned) Add(_ context.Context, userID int64, word string) error {
	if !slices.Contains(f.words[userID], word) {
		f.words[userID] = append(f.words[userID], word)
	}

	return nil
}

func (f *fakeLearned) Remove(_ context.Context, userID int64, word string) error {
	f.words[userID] = slices.DeleteFunc(f.words[userID], func(w string) bool { return w == word })

	return nil
}

type fixture struct {
	users    *fakeUsers
	sessions *fakeSessions
	learned  *fakeLearned
	metrics  *metrics.Metrics
	service  *practice.Service
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	catalog, err := accent.NewCatalog(sample())
	require.NoError(t, err)

	f := &fixture{
		users:    &fakeUsers{users: map[int64]*user.User{1: {ID: 1}}},
		sessions: &fakeSessions{sessions: map[int64]*practice.Session{}},
		learned:  &fakeLearned{words: map[int64][]string{}},
		metrics:  metrics.New(),
	}
	f.service = practice.NewService(f.sessions, f.learned, catalog, f.metrics)

	return f
}

func sample() []accent.Accent {
	return []accent.Accent{
		{ID: 1, Word: "алфАвІт", Note: "(подвійний наголос)"},
		{ID: 2, Word: "вИгода", Note: "(користь)"},
		{ID: 3, Word: "де-Юре"},
		{ID: 4, Word: "фОльга"},
	}
}

func TestBegin(main *testing.T) {
	main.Run("StartsFresh", func(t *testing.T) {
		f := newFixture(t)

		s, err := f.service.Begin(t.Context(), 1, 12)
		require.NoError(t, err)
		require.Equal(t, 12, s.Size)
		require.Zero(t, s.Answered)
		require.EqualValues(t, 1, f.metrics.SessionsStarted.Load())
	})

	main.Run("ReplacesRunInProgress", func(t *testing.T) {
		f := newFixture(t)

		first, err := f.service.Begin(t.Context(), 1, 12)
		require.NoError(t, err)
		first.Ask("фОльга")
		_, _ = first.Answer("фОльга")
		require.NoError(t, f.sessions.Save(t.Context(), first))

		second, err := f.service.Begin(t.Context(), 1, 24)
		require.NoError(t, err)
		require.Equal(t, 24, second.Size)
		require.Zero(t, second.Answered)
		require.Empty(t, second.AskedWords)
	})

	main.Run("RejectsUnsupportedLength", func(t *testing.T) {
		f := newFixture(t)

		_, err := f.service.Begin(t.Context(), 1, 13)
		require.Error(t, err)
	})
}

func TestQuestion(main *testing.T) {
	main.Run("NeverRepeatsAWord", func(t *testing.T) {
		f := newFixture(t)
		_, err := f.service.Begin(t.Context(), 1, 12)
		require.NoError(t, err)

		seen := map[string]struct{}{}
		for range len(sample()) {
			q, qErr := f.service.Question(t.Context(), 1)
			require.NoError(t, qErr)
			require.NotContains(t, seen, q.Word)
			seen[q.Word] = struct{}{}

			s, _ := f.sessions.Load(t.Context(), 1)
			_, _ = s.Answer(s.CurrentWord)
			require.NoError(t, f.sessions.Save(t.Context(), s))
		}
	})

	main.Run("EndsWhenTheListRunsOut", func(t *testing.T) {
		f := newFixture(t)
		_, err := f.service.Begin(t.Context(), 1, 12)
		require.NoError(t, err)

		for range len(sample()) {
			_, qErr := f.service.Question(t.Context(), 1)
			require.NoError(t, qErr)

			s, _ := f.sessions.Load(t.Context(), 1)
			// Answered correctly on purpose: a missed word goes back in the
			// pool, so wrong answers would never exhaust it.
			_, _ = s.Answer(s.CurrentWord)
			require.NoError(t, f.sessions.Save(t.Context(), s))
		}

		_, err = f.service.Question(t.Context(), 1)
		require.ErrorIs(t, err, practice.ErrOutOfWords)
	})

	main.Run("CountsAndNumbers", func(t *testing.T) {
		f := newFixture(t)
		_, err := f.service.Begin(t.Context(), 1, 12)
		require.NoError(t, err)

		q, err := f.service.Question(t.Context(), 1)
		require.NoError(t, err)
		require.Equal(t, 1, q.Number)
		require.Equal(t, 12, q.Total)
		require.NotEmpty(t, q.Variants)
		require.Contains(t, q.Variants, correctOf(t, f))
	})

	main.Run("HidesTheNoteOfADoubleStressedWord", func(t *testing.T) {
		f := newFixture(t)
		_, err := f.service.Begin(t.Context(), 1, 12)
		require.NoError(t, err)

		// Draw until the double-stressed word comes up.
		for range len(sample()) {
			q, qErr := f.service.Question(t.Context(), 1)
			require.NoError(t, qErr)

			if q.Word == "алфавіт" {
				require.Empty(t, q.Note, "naming the double stress would give the answer away")
				require.Contains(t, q.Variants, "алфАвІт")

				return
			}

			s, _ := f.sessions.Load(t.Context(), 1)
			// Answered correctly on purpose: a missed word goes back in the
			// pool, so wrong answers would never exhaust it.
			_, _ = s.Answer(s.CurrentWord)
			require.NoError(t, f.sessions.Save(t.Context(), s))
		}

		t.Fatal("the double-stressed word was never drawn")
	})

	main.Run("ShowsTheNoteOfAHomograph", func(t *testing.T) {
		f := newFixture(t)
		_, err := f.service.Begin(t.Context(), 1, 12)
		require.NoError(t, err)

		for range len(sample()) {
			q, qErr := f.service.Question(t.Context(), 1)
			require.NoError(t, qErr)

			if q.Word == "вигода" {
				require.Equal(t, "(користь)", q.Note)

				return
			}

			s, _ := f.sessions.Load(t.Context(), 1)
			// Answered correctly on purpose: a missed word goes back in the
			// pool, so wrong answers would never exhaust it.
			_, _ = s.Answer(s.CurrentWord)
			require.NoError(t, f.sessions.Save(t.Context(), s))
		}

		t.Fatal("the homograph was never drawn")
	})

	main.Run("NoSession", func(t *testing.T) {
		f := newFixture(t)

		_, err := f.service.Question(t.Context(), 1)
		require.ErrorIs(t, err, practice.ErrNoSession)
	})
}

func TestAnswerService(main *testing.T) {
	main.Run("CorrectAnswerLearnsTheWord", func(t *testing.T) {
		f := newFixture(t)
		q := ask(t, f)

		result, err := f.service.Answer(t.Context(), 1, correctOf(t, f))
		require.NoError(t, err)
		require.True(t, result.Correct)
		require.EqualValues(t, 1, f.metrics.AnswersCorrect.Load())
		require.Equal(t, []string{result.Word}, f.learned.words[1])
		require.NotEmpty(t, q.Variants)
	})

	main.Run("WrongAnswerUnlearnsTheWord", func(t *testing.T) {
		f := newFixture(t)
		ask(t, f)
		word := correctOf(t, f)
		f.learned.words[1] = []string{word}

		result, err := f.service.Answer(t.Context(), 1, missAt(word))
		require.NoError(t, err)
		require.False(t, result.Correct)
		require.Equal(t, word, result.Word, "the reply reveals the correct spelling")
		require.Empty(t, f.learned.words[1])
		require.EqualValues(t, 1, f.metrics.AnswersWrong.Load())
	})

	main.Run("ReportsRunCompletion", func(t *testing.T) {
		f := newFixture(t)
		_, err := f.service.Begin(t.Context(), 1, 12)
		require.NoError(t, err)

		s, _ := f.sessions.Load(t.Context(), 1)
		s.Answered = 11
		s.Correct = 11
		require.NoError(t, f.sessions.Save(t.Context(), s))

		_, err = f.service.Question(t.Context(), 1)
		require.NoError(t, err)

		result, err := f.service.Answer(t.Context(), 1, correctOf(t, f))
		require.NoError(t, err)
		require.True(t, result.RunComplete)
	})

	main.Run("NoQuestionOutstanding", func(t *testing.T) {
		f := newFixture(t)
		_, err := f.service.Begin(t.Context(), 1, 12)
		require.NoError(t, err)

		_, err = f.service.Answer(t.Context(), 1, "будь-що")
		require.ErrorIs(t, err, practice.ErrStaleAnswer)
	})

	main.Run("AnswerToAnotherWordIsIgnored", func(t *testing.T) {
		// A button from a question already dealt with, or one tapped twice
		// before the first tap landed.
		f := newFixture(t)
		ask(t, f)

		_, err := f.service.Answer(t.Context(), 1, "зовсІм інше слово")
		require.ErrorIs(t, err, practice.ErrStaleAnswer)

		s, err := f.sessions.Load(t.Context(), 1)
		require.NoError(t, err)
		require.Zero(t, s.Answered, "a stale tap must not count")
		require.NotEmpty(t, s.CurrentWord, "the question stays open")
	})

	main.Run("TappingTwiceCountsOnce", func(t *testing.T) {
		f := newFixture(t)
		ask(t, f)
		word := correctOf(t, f)

		_, err := f.service.Answer(t.Context(), 1, word)
		require.NoError(t, err)

		_, err = f.service.Answer(t.Context(), 1, word)
		require.ErrorIs(t, err, practice.ErrStaleAnswer)

		s, err := f.sessions.Load(t.Context(), 1)
		require.NoError(t, err)
		require.Equal(t, 1, s.Answered)
	})
}

func TestFinish(main *testing.T) {
	main.Run("GradesACompletedRun", func(t *testing.T) {
		f := newFixture(t)

		_, err := f.service.Begin(t.Context(), 1, 12)
		require.NoError(t, err)

		s, _ := f.sessions.Load(t.Context(), 1)
		s.Answered, s.Correct = 12, 12
		require.NoError(t, f.sessions.Save(t.Context(), s))

		summary, err := f.service.Finish(t.Context(), 1)
		require.NoError(t, err)
		require.True(t, summary.Completed)
		require.Equal(t, practice.GradeExcellent, summary.Grade)
		require.EqualValues(t, 1, f.metrics.SessionsCompleted.Load())
	})

	main.Run("AbandonedRunIsNotCompleted", func(t *testing.T) {
		f := newFixture(t)

		_, err := f.service.Begin(t.Context(), 1, 12)
		require.NoError(t, err)

		summary, err := f.service.Finish(t.Context(), 1)
		require.NoError(t, err)
		require.False(t, summary.Completed)
		require.EqualValues(t, 1, f.metrics.SessionsAbandoned.Load())
	})

	main.Run("NoSession", func(t *testing.T) {
		f := newFixture(t)

		_, err := f.service.Finish(t.Context(), 1)
		require.ErrorIs(t, err, practice.ErrNoSession)
	})
}

func TestMissedWords(main *testing.T) {
	// The old schema has nowhere to keep these, so they live in the service
	// for the length of a run and the results name them.
	answer := func(t *testing.T, f *fixture, correctly bool) string {
		t.Helper()

		_, err := f.service.Question(t.Context(), 1)
		require.NoError(t, err)

		s, err := f.sessions.Load(t.Context(), 1)
		require.NoError(t, err)
		word := s.CurrentWord

		given := word
		if !correctly {
			given = missAt(word)
		}

		_, err = f.service.Answer(t.Context(), 1, given)
		require.NoError(t, err)

		return word
	}

	main.Run("ReportsWhatWasMissed", func(t *testing.T) {
		f := newFixture(t)
		_, err := f.service.Begin(t.Context(), 1, 12)
		require.NoError(t, err)

		missed := answer(t, f, false)
		answer(t, f, true)

		summary, err := f.service.Finish(t.Context(), 1)
		require.NoError(t, err)
		require.Equal(t, []string{missed}, summary.Missed)
	})

	main.Run("NothingMissedIsEmpty", func(t *testing.T) {
		f := newFixture(t)
		_, err := f.service.Begin(t.Context(), 1, 12)
		require.NoError(t, err)

		answer(t, f, true)

		summary, err := f.service.Finish(t.Context(), 1)
		require.NoError(t, err)
		require.Empty(t, summary.Missed)
	})

	main.Run("ANewRunStartsClean", func(t *testing.T) {
		f := newFixture(t)
		_, err := f.service.Begin(t.Context(), 1, 12)
		require.NoError(t, err)
		answer(t, f, false)

		_, err = f.service.Begin(t.Context(), 1, 12)
		require.NoError(t, err)
		answer(t, f, true)

		summary, err := f.service.Finish(t.Context(), 1)
		require.NoError(t, err)
		require.Empty(t, summary.Missed, "misses from the previous run leaked into this one")
	})

	main.Run("AbandoningClearsThem", func(t *testing.T) {
		f := newFixture(t)
		_, err := f.service.Begin(t.Context(), 1, 12)
		require.NoError(t, err)
		answer(t, f, false)

		require.NoError(t, f.service.Abandon(t.Context(), 1))

		_, err = f.service.Begin(t.Context(), 1, 12)
		require.NoError(t, err)
		answer(t, f, true)

		summary, err := f.service.Finish(t.Context(), 1)
		require.NoError(t, err)
		require.Empty(t, summary.Missed)
	})

	main.Run("AWordMissedTwiceIsNamedOnce", func(t *testing.T) {
		f := newFixture(t)
		_, err := f.service.Begin(t.Context(), 1, 12)
		require.NoError(t, err)

		// A missed word goes back in the pool, so with a small catalog it
		// comes round again quickly.
		seen := map[string]int{}
		for range 12 {
			seen[answer(t, f, false)]++
		}

		summary, err := f.service.Finish(t.Context(), 1)
		require.NoError(t, err)

		var repeated bool
		for _, n := range seen {
			if n > 1 {
				repeated = true
			}
		}
		require.True(t, repeated, "a missed word never came back")
		require.Len(t, summary.Missed, len(seen), "each word should be named once")
	})
}

func TestAbandon(t *testing.T) {
	f := newFixture(t)
	_, err := f.service.Begin(t.Context(), 1, 12)
	require.NoError(t, err)

	require.NoError(t, f.service.Abandon(t.Context(), 1))

	_, err = f.sessions.Load(t.Context(), 1)
	require.ErrorIs(t, err, practice.ErrNoSession)
}

// ask starts a run and draws its first question.
func ask(t *testing.T, f *fixture) practice.Question {
	t.Helper()

	_, err := f.service.Begin(t.Context(), 1, 12)
	require.NoError(t, err)

	q, err := f.service.Question(t.Context(), 1)
	require.NoError(t, err)

	return q
}

// correctOf returns the spelling that answers the outstanding question.
func correctOf(t *testing.T, f *fixture) string {
	t.Helper()

	s, err := f.sessions.Load(t.Context(), 1)
	require.NoError(t, err)

	return s.CurrentWord
}

// missAt returns a wrong stress of the same word: the plain spelling, which
// belongs to the open question but is not the answer to it.
func missAt(word string) string {
	return words.Normalize(word)
}
