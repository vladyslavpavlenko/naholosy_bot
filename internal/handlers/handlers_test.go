package handlers_test

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mymmrac/telego"
	"github.com/stretchr/testify/require"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/accent"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/broadcast"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/handlers"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/metrics"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/practice"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/responses"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/sender"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/stats"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/storage/sqlite"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/user"
	"github.com/vladyslavpavlenko/naholosy_bot/pkg/logger"
)

// sent is one message the bot produced.
type sent struct {
	text     string
	document bool
	markup   telego.ReplyMarkup
}

// quizSent is one question put to the user as a poll.
type quizSent struct {
	pollID      string
	openPeriod  time.Duration
	question    string
	description string
	options     []string
	correct     int
	explanation string
}

// recorder stands in for Telegram and keeps what the bot tried to send.
type recorder struct {
	mu       sync.Mutex
	messages []sent
	quizzes  []quizSent
}

func (r *recorder) Send(_ context.Context, m sender.Message) error {
	return r.add(sent{text: m.Text, markup: m.Markup})
}

func (r *recorder) SendQuiz(_ context.Context, q sender.Quiz) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := fmt.Sprintf("poll-%d", len(r.quizzes)+1)
	r.quizzes = append(r.quizzes, quizSent{
		pollID:      id,
		openPeriod:  q.OpenPeriod,
		question:    q.Question,
		description: q.Description,
		options:     q.Options,
		correct:     q.Correct,
		explanation: q.Explanation,
	})

	return id, nil
}

// openQuiz returns the question currently put to the user.
func (r *recorder) openQuiz(t *testing.T) quizSent {
	t.Helper()

	r.mu.Lock()
	defer r.mu.Unlock()
	require.NotEmpty(t, r.quizzes, "no question was asked")

	return r.quizzes[len(r.quizzes)-1]
}

func (r *recorder) SendDocument(context.Context, int64, telego.InputFile) (string, error) {
	return "cached-file-id", r.add(sent{document: true})
}

func (r *recorder) CopyMessage(context.Context, int64, int64, int) error {
	return r.add(sent{text: "<copy>"})
}

func (r *recorder) add(s sent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.messages = append(r.messages, s)

	return nil
}

// drain returns everything sent since the last call.
func (r *recorder) drain() []sent {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := r.messages
	r.messages = nil

	return out
}

// last returns the most recent message.
func (r *recorder) last(t *testing.T) sent {
	t.Helper()

	r.mu.Lock()
	defer r.mu.Unlock()
	require.NotEmpty(t, r.messages)

	return r.messages[len(r.messages)-1]
}

type harness struct {
	handlers *handlers.Handlers
	sent     *recorder
	users    *sqlite.Users
	sessions *sqlite.Sessions
	learned  *sqlite.LearnedWords
	catalog  *accent.Catalog
}

// newHarness builds the real stack over a temporary database, with Telegram
// replaced by a recorder.
func newHarness(t *testing.T) *harness {
	t.Helper()

	return newHarnessWith(t, &recorder{})
}

// newHarnessWith is newHarness with a sender of the caller's choosing.
func newHarnessWith(t *testing.T, rec *recorder) *harness {
	t.Helper()

	db, err := sqlite.Open(t.Context(), filepath.Join(t.TempDir(), "naholosy.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	list, err := sqlite.NewAccents(db).All(t.Context())
	require.NoError(t, err)

	catalog, err := accent.NewCatalog(list)
	require.NoError(t, err)

	var (
		m       = metrics.New()
		log     = logger.New("PROD")
		users   = sqlite.NewUsers(db)
		session = sqlite.NewSessions(db)
		learned = sqlite.NewLearnedWords(db)
	)

	h := handlers.New(
		users,
		practice.NewService(session, learned, catalog, m),
		stats.NewService(sqlite.NewStats(db), m, catalog),
		broadcast.NewService(users, &broadcastSender{rec}, m, log),
		catalog,
		rec,
		m,
		log,
	)
	return &harness{handlers: h, sent: rec, users: users, sessions: session, learned: learned, catalog: catalog}
}

// broadcastSender adapts the recorder to the broadcast service's port.
type broadcastSender struct{ rec *recorder }

func (b *broadcastSender) SendText(ctx context.Context, userID int64, text string) error {
	return b.rec.Send(ctx, sender.Message{ChatID: userID, Text: text})
}

func (b *broadcastSender) CopyMessage(ctx context.Context, userID, from int64, id int) error {
	return b.rec.CopyMessage(ctx, userID, from, id)
}

// request builds a request for a user, loading their current state.
func (h *harness) request(t *testing.T, text string) handlers.Request {
	t.Helper()

	u, err := h.users.Ensure(t.Context(), 1)
	require.NoError(t, err)

	return handlers.Request{ChatID: 1, MessageID: 100, Text: text, User: u}
}

// pick builds the request answering the open poll with the option at index.
func (h *harness) pick(t *testing.T, index int) handlers.Request {
	t.Helper()

	return h.answer(t, h.sent.openQuiz(t).pollID, index)
}

// answer builds a poll answer for an arbitrary poll.
func (h *harness) answer(t *testing.T, pollID string, index int) handlers.Request {
	t.Helper()

	u, err := h.users.Ensure(t.Context(), 1)
	require.NoError(t, err)

	return handlers.Request{
		ChatID: 1,
		User:   u,
		Poll:   &handlers.PollAnswer{PollID: pollID, Options: []int{index}},
	}
}

// expire builds the request a question running out of time produces.
func (h *harness) expire(t *testing.T, pollID string) handlers.Request {
	t.Helper()

	u, err := h.users.Ensure(t.Context(), 1)
	require.NoError(t, err)

	return handlers.Request{ChatID: 1, User: u, Poll: &handlers.PollAnswer{PollID: pollID}}
}

// stage reads the user's persisted stage.
func (h *harness) stage(t *testing.T) user.Stage {
	t.Helper()

	u, err := h.users.Get(t.Context(), 1)
	require.NoError(t, err)

	return u.Stage
}

func TestStartAndMenu(main *testing.T) {
	main.Run("Start", func(t *testing.T) {
		h := newHarness(t)

		require.NoError(t, h.handlers.Start(t.Context(), h.request(t, "/start")))
		require.Equal(t, responses.Start, h.sent.last(t).text)
		require.Equal(t, user.StageStart, h.stage(t))
	})

	main.Run("MainMenu", func(t *testing.T) {
		h := newHarness(t)

		require.NoError(t, h.handlers.MainMenu(t.Context(), h.request(t, responses.FasterButton)))
		require.Equal(t, responses.MainMenu, h.sent.last(t).text)
		require.Equal(t, user.StageMainMenu, h.stage(t))
	})

	main.Run("DownloadCachesTheFileID", func(t *testing.T) {
		h := newHarness(t)

		require.NoError(t, h.handlers.Download(t.Context(), h.request(t, responses.DownloadButton)))
		require.NoError(t, h.handlers.Download(t.Context(), h.request(t, responses.DownloadButton)))

		messages := h.sent.drain()
		require.Len(t, messages, 2)
		require.True(t, messages[0].document)
		require.True(t, messages[1].document)
	})
}

func TestWords(main *testing.T) {
	main.Run("LookupFound", func(t *testing.T) {
		h := newHarness(t)

		require.NoError(t, h.handlers.Lookup(t.Context(), h.request(t, "фольга")))
		require.Contains(t, h.sent.last(t).text, "фОльга")
	})

	main.Run("LookupIsCaseAndApostropheInsensitive", func(t *testing.T) {
		h := newHarness(t)

		require.NoError(t, h.handlers.Lookup(t.Context(), h.request(t, "ТІМ’ЯНИЙ")))
		require.Contains(t, h.sent.last(t).text, "тім'янИй")
	})

	main.Run("LookupHomographsShowsBoth", func(t *testing.T) {
		h := newHarness(t)

		require.NoError(t, h.handlers.Lookup(t.Context(), h.request(t, "вигода")))
		text := h.sent.last(t).text
		require.Contains(t, text, "вИгода")
		require.Contains(t, text, "вигОда")
	})

	main.Run("LookupMissingSaysSo", func(t *testing.T) {
		h := newHarness(t)

		// The old bot sent an empty message here, which Telegram rejects.
		require.NoError(t, h.handlers.Lookup(t.Context(), h.request(t, "будинок")))
		require.Equal(t, responses.WordNotFound, h.sent.last(t).text)
	})

	main.Run("SearchByLetters", func(t *testing.T) {
		h := newHarness(t)

		require.NoError(t, h.handlers.SearchByLetters(t.Context(), h.request(t, "є я")))
		text := h.sent.last(t).text
		require.Contains(t, text, "Є Я")
		require.Contains(t, text, "єретИк")
		require.Contains(t, text, "ярмаркОвий")
		require.NotContains(t, text, "фОльга")
	})

	main.Run("SearchByLetterWithNoWords", func(t *testing.T) {
		h := newHarness(t)

		require.NoError(t, h.handlers.SearchByLetters(t.Context(), h.request(t, "ю")))
		require.Contains(t, h.sent.last(t).text, responses.NoWordsForLetters)
	})

	main.Run("WordsMenu", func(t *testing.T) {
		h := newHarness(t)

		require.NoError(t, h.handlers.WordsMenu(t.Context(), h.request(t, responses.AllWordsButton)))
		require.Equal(t, responses.WordsMenu, h.sent.last(t).text)
		require.Equal(t, user.StageWords, h.stage(t))
	})
}

func TestPracticeFlow(main *testing.T) {
	main.Run("TutorialIsShownOnce", func(t *testing.T) {
		h := newHarness(t)
		require.NoError(t, h.handlers.PracticeMenu(t.Context(), h.request(t, responses.PracticeButton)))
		h.sent.drain()

		require.NoError(t, h.handlers.ChooseSize(t.Context(), h.request(t, "12")))
		first := h.sent.drain()
		require.Len(t, first, 3, "the rules are explained across three messages")
		require.Contains(t, first[0].text, "12")
		require.Equal(t, responses.PracticeExplanation3, first[2].text)
		require.Equal(t, user.StagePracticeReadiness, h.stage(t))

		require.NoError(t, h.handlers.PracticeMenu(t.Context(), h.request(t, responses.BackButton)))
		h.sent.drain()

		require.NoError(t, h.handlers.ChooseSize(t.Context(), h.request(t, "24")))
		second := h.sent.drain()
		require.Len(t, second, 1, "the rules are not repeated")
		require.Equal(t, responses.PracticeExplanation3, second[0].text)
	})

	main.Run("RunsToCompletion", func(t *testing.T) {
		h := newHarness(t)
		const size = 12

		require.NoError(t, h.handlers.PracticeMenu(t.Context(), h.request(t, responses.PracticeButton)))
		require.NoError(t, h.handlers.ChooseSize(t.Context(), h.request(t, strconv.Itoa(size))))
		require.NoError(t, h.handlers.StartRun(t.Context(), h.request(t, responses.YesButton)))
		require.Equal(t, user.StageGame, h.stage(t))

		first := h.sent.openQuiz(t)
		require.Equal(t, "1 / 12", first.description, "the counter belongs in the description")
		require.NotContains(t, first.question, "/", "the question is nothing but the word")

		// A missed word goes back in the pool, so which words come up is not
		// fixed. Track what the run actually did and check against that.
		expectLearned := map[string]struct{}{}
		var missed string

		for i := range size {
			quiz := h.sent.openQuiz(t)
			word := quiz.options[quiz.correct]

			// Answer everything correctly except the third question.
			pick := quiz.correct
			if i == 2 {
				pick = (quiz.correct + 1) % len(quiz.options)
				missed = word
				delete(expectLearned, word)
			} else {
				expectLearned[word] = struct{}{}
			}

			require.NoError(t, h.handlers.Answer(t.Context(), h.pick(t, pick)))
		}

		// Telegram marks the right and wrong option itself, so the bot says
		// nothing about the verdict: one poll per question and no more.
		require.Len(t, h.sent.quizzes, size)

		messages := h.sent.drain()
		texts := make([]string, 0, len(messages))
		for _, m := range messages {
			texts = append(texts, m.text)
		}
		joined := strings.Join(texts, "\n")

		require.Contains(t, joined, "🏁", "the run is reported")
		require.Contains(t, joined, "<b>12 / 12</b>")
		require.Contains(t, joined, "<b>11</b>")
		require.Contains(t, joined, "Варто повторити", "the results name what to go back to")
		require.Contains(t, joined, missed)
		require.Contains(t, joined, responses.PracticeMenu, "the user lands back in the practice menu")
		require.Equal(t, user.StagePracticeMenu, h.stage(t))

		learned, err := h.learned.List(t.Context(), 1)
		require.NoError(t, err)
		require.ElementsMatch(t, keysOf(expectLearned), learned,
			"only the words answered correctly are learned")
	})

	main.Run("TheCorrectOptionIsMarkedForTelegram", func(t *testing.T) {
		h := newHarness(t)

		require.NoError(t, h.handlers.PracticeMenu(t.Context(), h.request(t, responses.PracticeButton)))
		require.NoError(t, h.handlers.ChooseSize(t.Context(), h.request(t, "12")))
		require.NoError(t, h.handlers.StartRun(t.Context(), h.request(t, responses.YesButton)))

		quiz := h.sent.openQuiz(t)
		require.GreaterOrEqual(t, quiz.correct, 0)
		require.Less(t, quiz.correct, len(quiz.options))

		s, err := h.sessions.Load(t.Context(), 1)
		require.NoError(t, err)
		require.Equal(t, s.CurrentWord, quiz.options[quiz.correct],
			"Telegram would highlight the wrong option")

		// Telegram takes 2 to 12 options and caps the explanation at 200.
		require.GreaterOrEqual(t, len(quiz.options), 2)
		require.LessOrEqual(t, len(quiz.options), 12)
		require.LessOrEqual(t, len([]rune(quiz.explanation)), 200)
		require.Contains(t, quiz.explanation, s.CurrentWord)
	})

	main.Run("AnswerToAPollFromAnEarlierProcessMovesOn", func(t *testing.T) {
		// The open polls live in memory, so a restart leaves the user looking
		// at a question nothing can grade. Asking the next one beats silence.
		h := newHarness(t)

		require.NoError(t, h.handlers.PracticeMenu(t.Context(), h.request(t, responses.PracticeButton)))
		require.NoError(t, h.handlers.ChooseSize(t.Context(), h.request(t, "12")))
		require.NoError(t, h.handlers.StartRun(t.Context(), h.request(t, responses.YesButton)))
		asked := len(h.sent.quizzes)

		require.NoError(t, h.handlers.Answer(t.Context(), h.answer(t, "poll-from-before", 0)))

		require.Len(t, h.sent.quizzes, asked+1, "a fresh question should have been asked")

		s, err := h.sessions.Load(t.Context(), 1)
		require.NoError(t, err)
		require.Zero(t, s.Answered, "an ungradeable answer must not be scored")
	})

	main.Run("QuestionsCarryATimeout", func(t *testing.T) {
		h := newHarness(t)
		h.startRun(t)

		require.Equal(t, 7*time.Second, h.sent.openQuiz(t).openPeriod)
	})

	main.Run("OneQuestionRunningOutCarriesOn", func(t *testing.T) {
		h := newHarness(t)
		h.startRun(t)
		asked := len(h.sent.quizzes)

		require.NoError(t, h.handlers.Timeout(t.Context(), h.expire(t, h.sent.openQuiz(t).pollID)))

		require.Len(t, h.sent.quizzes, asked+1, "the next question should have been asked")
		require.Equal(t, user.StageGame, h.stage(t))
	})

	main.Run("TwoInARowStopTheRun", func(t *testing.T) {
		h := newHarness(t)
		h.startRun(t)

		require.NoError(t, h.handlers.Timeout(t.Context(), h.expire(t, h.sent.openQuiz(t).pollID)))
		require.NoError(t, h.handlers.Timeout(t.Context(), h.expire(t, h.sent.openQuiz(t).pollID)))

		texts := h.texts()
		require.Contains(t, texts, "Зупиняю тренування")
		require.Contains(t, texts, "🏁", "the tally is still reported")
		require.Contains(t, texts, responses.PracticeMenu)
		require.Equal(t, user.StagePracticeMenu, h.stage(t))
	})

	main.Run("AnsweringResetsTheRunOfMisses", func(t *testing.T) {
		h := newHarness(t)
		h.startRun(t)

		require.NoError(t, h.handlers.Timeout(t.Context(), h.expire(t, h.sent.openQuiz(t).pollID)))

		quiz := h.sent.openQuiz(t)
		require.NoError(t, h.handlers.Answer(t.Context(), h.pick(t, quiz.correct)))

		// The miss before the answer must not count towards stopping.
		require.NoError(t, h.handlers.Timeout(t.Context(), h.expire(t, h.sent.openQuiz(t).pollID)))

		require.NotContains(t, h.texts(), "Зупиняю тренування")
		require.Equal(t, user.StageGame, h.stage(t))
	})

	main.Run("AnAnsweredQuestionRunningOutIsIgnored", func(t *testing.T) {
		// The poll still closes at the end of its period even once answered.
		h := newHarness(t)
		h.startRun(t)

		quiz := h.sent.openQuiz(t)
		require.NoError(t, h.handlers.Answer(t.Context(), h.pick(t, quiz.correct)))
		asked := len(h.sent.quizzes)

		require.NoError(t, h.handlers.Timeout(t.Context(), h.expire(t, quiz.pollID)))

		require.Len(t, h.sent.quizzes, asked, "no extra question")
		require.NotContains(t, h.texts(), "Зупиняю тренування")
	})

	main.Run("GivingUpReportsWhatWasDone", func(t *testing.T) {
		h := newHarness(t)

		require.NoError(t, h.handlers.PracticeMenu(t.Context(), h.request(t, responses.PracticeButton)))
		require.NoError(t, h.handlers.ChooseSize(t.Context(), h.request(t, "36")))
		require.NoError(t, h.handlers.StartRun(t.Context(), h.request(t, responses.YesButton)))

		quiz := h.sent.openQuiz(t)
		require.NoError(t, h.handlers.Answer(t.Context(), h.pick(t, quiz.correct)))
		h.sent.drain()

		require.NoError(t, h.handlers.FinishRun(t.Context(), h.request(t, responses.FinishGameButton)))

		messages := h.sent.drain()
		require.Contains(t, messages[0].text, "<b>1 / 36</b>")
		require.Equal(t, user.StagePracticeMenu, h.stage(t))
	})

	main.Run("AnswerWithoutARunReturnsToTheMenu", func(t *testing.T) {
		h := newHarness(t)

		require.NoError(t, h.handlers.Answer(t.Context(), h.answer(t, "poll-1", 0)))
		require.Equal(t, responses.PracticeMenu, h.sent.last(t).text)
		require.Equal(t, user.StagePracticeMenu, h.stage(t))
	})

	main.Run("PracticeMenuClearsAnUnfinishedRun", func(t *testing.T) {
		h := newHarness(t)

		require.NoError(t, h.handlers.PracticeMenu(t.Context(), h.request(t, responses.PracticeButton)))
		require.NoError(t, h.handlers.ChooseSize(t.Context(), h.request(t, "12")))
		require.NoError(t, h.handlers.StartRun(t.Context(), h.request(t, responses.YesButton)))
		require.NoError(t, h.handlers.PracticeMenu(t.Context(), h.request(t, responses.PracticeButton)))

		s, err := h.sessions.Load(t.Context(), 1)
		require.NoError(t, err)
		require.Zero(t, s.Answered)
		require.Empty(t, s.CurrentWord)
		require.Empty(t, s.AskedWords)
	})
}

func TestStatus(t *testing.T) {
	h := newHarness(t)

	require.NoError(t, h.handlers.Status(t.Context(), h.request(t, "/status")))
	text := h.sent.last(t).text
	require.Contains(t, text, "📊")
	require.Contains(t, text, strconv.Itoa(h.catalog.Len()))
}

func TestBroadcast(main *testing.T) {
	main.Run("PreparesThenConfirms", func(t *testing.T) {
		h := newHarness(t)

		req := h.request(t, "/broadcast привіт")
		req.Args = "привіт"

		require.NoError(t, h.handlers.BroadcastPrepare(t.Context(), req))
		prepared := h.sent.drain()
		require.Len(t, prepared, 2, "a preview and a confirmation prompt")
		require.Equal(t, "привіт", prepared[0].text)
		require.Contains(t, prepared[1].text, "/broadcast_confirm")
	})

	main.Run("WithoutTextShowsUsage", func(t *testing.T) {
		h := newHarness(t)

		require.NoError(t, h.handlers.BroadcastPrepare(t.Context(), h.request(t, "/broadcast")))
		require.Equal(t, responses.BroadcastUsage, h.sent.last(t).text)
	})

	main.Run("ReplyBecomesACopy", func(t *testing.T) {
		h := newHarness(t)

		req := h.request(t, "/broadcast")
		req.ReplyTo = &handlers.Reply{ChatID: 1, MessageID: 55}

		require.NoError(t, h.handlers.BroadcastPrepare(t.Context(), req))
		require.Equal(t, "<copy>", h.sent.drain()[0].text)
	})

	main.Run("TestGoesOnlyToTheAdmin", func(t *testing.T) {
		h := newHarness(t)

		req := h.request(t, "/broadcast_test привіт")
		req.Args = "привіт"

		require.NoError(t, h.handlers.BroadcastTest(t.Context(), req))
		messages := h.sent.drain()
		require.Len(t, messages, 2)
		require.Equal(t, "привіт", messages[0].text)
		require.Equal(t, responses.BroadcastTestSent, messages[1].text)
	})

	main.Run("ConfirmWithoutPreparation", func(t *testing.T) {
		h := newHarness(t)

		require.NoError(t, h.handlers.BroadcastConfirm(t.Context(), h.request(t, "/broadcast_confirm")))
		require.Equal(t, responses.BroadcastNothing, h.sent.last(t).text)
	})

	main.Run("Cancel", func(t *testing.T) {
		h := newHarness(t)

		req := h.request(t, "/broadcast привіт")
		req.Args = "привіт"
		require.NoError(t, h.handlers.BroadcastPrepare(t.Context(), req))
		h.sent.drain()

		require.NoError(t, h.handlers.BroadcastCancel(t.Context(), h.request(t, "/broadcast_cancel")))
		require.Equal(t, responses.BroadcastCanceled, h.sent.last(t).text)
	})
}

// barrierSender releases every caller of SendDocument at the same instant and
// synchronizes nothing else, so that the writes to the cached file ID really
// do overlap. A sender that took a lock would order them and hide the race.
type barrierSender struct {
	recorder

	entered sync.WaitGroup
	release chan struct{}
}

func (b *barrierSender) SendDocument(context.Context, int64, telego.InputFile) (string, error) {
	b.entered.Done()
	<-b.release

	return "cached-file-id", nil
}

// TestDownloadIsSafeConcurrently covers the cached file ID, which is shared by
// every user while the dispatcher's lock is only per user. Run with -race.
func TestDownloadIsSafeConcurrently(t *testing.T) {
	const callers = 8

	barrier := &barrierSender{release: make(chan struct{})}
	barrier.entered.Add(callers)

	h := newHarnessWith(t, &barrier.recorder)
	h.handlers = handlers.New(h.users, nil, nil, nil, h.catalog, barrier, metrics.New(), logger.New("PROD"))

	u, err := h.users.Ensure(t.Context(), 1)
	require.NoError(t, err)
	req := handlers.Request{ChatID: 1, MessageID: 1, User: u}

	errs := make([]error, callers)

	var wg sync.WaitGroup
	for i := range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs[i] = h.handlers.Download(t.Context(), req)
		}()
	}

	barrier.entered.Wait()
	close(barrier.release)
	wg.Wait()

	for _, err := range errs {
		require.NoError(t, err)
	}
}

func keysOf(set map[string]struct{}) []string {
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}

	return keys
}

// panicSender blows up on the first delivery, standing in for anything that
// can go wrong deep inside background work.
type panicSender struct{ recorder }

func (p *panicSender) SendText(context.Context, int64, string) error {
	panic("something went wrong far from the update handler")
}

func (p *panicSender) CopyMessage(context.Context, int64, int64, int) error { return nil }

// TestBackgroundWorkSurvivesAPanic covers the one goroutine the bot starts on
// its own. The update handler's recovery does not reach it, so a panic in a
// broadcast would otherwise end the process.
func TestBackgroundWorkSurvivesAPanic(t *testing.T) {
	h := newHarness(t)

	blowUp := &panicSender{}
	m := metrics.New()
	handler := handlers.New(
		h.users,
		nil,
		nil,
		broadcast.NewService(h.users, blowUp, m, logger.New("PROD")),
		h.catalog,
		&blowUp.recorder,
		m,
		logger.New("PROD"),
	)

	u, err := h.users.Ensure(t.Context(), 1)
	require.NoError(t, err)
	req := handlers.Request{ChatID: 1, MessageID: 1, Text: "/broadcast привіт", Args: "привіт", User: u}

	require.NoError(t, handler.BroadcastPrepare(t.Context(), req))
	require.NoError(t, handler.BroadcastConfirm(t.Context(), req))

	// The goroutine outlives the call, so wait for it to have blown up and
	// been caught.
	require.Eventually(t, func() bool {
		return m.Snapshot().PanicsRecovered > 0
	}, 2*time.Second, 10*time.Millisecond, "the panic was not recovered")
}

// startRun takes a user from the practice menu to the first question.
func (h *harness) startRun(t *testing.T) {
	t.Helper()

	require.NoError(t, h.handlers.PracticeMenu(t.Context(), h.request(t, responses.PracticeButton)))
	require.NoError(t, h.handlers.ChooseSize(t.Context(), h.request(t, "12")))
	require.NoError(t, h.handlers.StartRun(t.Context(), h.request(t, responses.YesButton)))
}

// texts joins everything the bot has said so far.
func (h *harness) texts() string {
	h.sent.mu.Lock()
	defer h.sent.mu.Unlock()

	var b strings.Builder
	for _, m := range h.sent.messages {
		b.WriteString(m.text)
		b.WriteString("\n")
	}

	return b.String()
}

// TestQuestionsAnnounceTheirOwnTimeout covers the clock itself. Telegram
// closes an expired poll without saying so, so if this stops working a run
// simply hangs on the question, which is what it did before the clock existed.
func TestQuestionsAnnounceTheirOwnTimeout(main *testing.T) {
	main.Run("AnUnansweredQuestionIsAnnounced", func(t *testing.T) {
		h := newHarness(t)
		h.handlers.SetQuestionClock(20*time.Millisecond, 10*time.Millisecond)
		h.startRun(t)

		asked := h.sent.openQuiz(t)

		select {
		case timeout := <-h.handlers.Timeouts():
			require.Equal(t, asked.pollID, timeout.PollID)
			require.EqualValues(t, 1, timeout.UserID)
		case <-time.After(2 * time.Second):
			t.Fatal("the question never reported itself as expired")
		}
	})

	main.Run("AnAnsweredQuestionIsNot", func(t *testing.T) {
		h := newHarness(t)
		h.handlers.SetQuestionClock(20*time.Millisecond, 10*time.Millisecond)
		h.startRun(t)

		quiz := h.sent.openQuiz(t)
		require.NoError(t, h.handlers.Answer(t.Context(), h.pick(t, quiz.correct)))

		// The next question has its own clock, so only the answered one must
		// stay quiet.
		for {
			select {
			case timeout := <-h.handlers.Timeouts():
				require.NotEqual(t, quiz.pollID, timeout.PollID,
					"an answered question should not expire")
			case <-time.After(300 * time.Millisecond):
				return
			}
		}
	})
}
