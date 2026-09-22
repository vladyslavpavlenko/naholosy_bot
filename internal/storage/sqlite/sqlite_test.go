package sqlite_test

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/accent"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/practice"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/storage/sqlite"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/user"

	// Registers the driver the raw fixture below opens with.
	_ "modernc.org/sqlite"
)

// legacySchema is the schema the first version of the bot created, written out
// here so the tests exercise a database it really could have left behind.
const legacySchema = `
CREATE TABLE "naholosy" (
	"id" INTEGER, "letter" VARCHAR(1), "word_lowercase" VARCHAR(50),
	"word" VARCHAR(25), "hint" VARCHAR(50), PRIMARY KEY("id"));
CREATE TABLE "users" (
	"user_id" INTEGER UNIQUE, "menu_stage" VARCHAR(25), "skip_tutorial" INTEGER,
	"practice_mode" INTEGER, "already_asked_words" VARCHAR(700), "asked_word" VARCHAR(25),
	"correct_answers" INTEGER, "wrong_answers" INTEGER, "answered_count" INTEGER,
	"combo_count" INTEGER);
CREATE TABLE "users_xp" ("user_id" INTEGER UNIQUE, "learned_words" VARCHAR(15000));
`

// openFresh opens an empty database, which seeds itself with the word list.
func openFresh(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sqlite.Open(t.Context(), filepath.Join(t.TempDir(), "naholosy.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	return db
}

// openLegacy builds a database in the old shape, fills it with rows in the
// old bot's style, and opens it the way the bot would.
func openLegacy(t *testing.T) *sql.DB {
	t.Helper()

	path := filepath.Join(t.TempDir(), "naholosy.db")

	raw, err := sql.Open("sqlite", "file:"+path)
	require.NoError(t, err)

	_, err = raw.ExecContext(t.Context(), legacySchema)
	require.NoError(t, err)

	_, err = raw.ExecContext(t.Context(), `INSERT INTO naholosy (id, letter, word_lowercase, word, hint) VALUES
		(1, 'А', 'алфавіт', 'алфАвІт', '(слово має подвійний наголос)'),
		(2, 'Ф', 'фольга',  'фОльга',  ''),
		(3, 'В', 'вигода',  'вИгода',  '(користь)'),
		(4, 'В', 'вигода',  'вигОда',  '(зручність)'),
		(5, 'Т', 'тім’яний', 'тім’янИй', '')`)
	require.NoError(t, err)

	// A user mid-run, a user who never got past the greeting, and a row with
	// the NULLs and malformed JSON the old bot really wrote.
	_, err = raw.ExecContext(t.Context(), `INSERT INTO users VALUES
		(1, 'game', 1, 12, '["фОльга"]', 'вИгода', 3, 1, 4, 2),
		(2, 'start', NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL),
		(3, 'practice_menu', 1, 12, '{[]}', '{[]}', 0, 0, 0, 0)`)
	require.NoError(t, err)

	_, err = raw.ExecContext(t.Context(), `INSERT INTO users_xp VALUES (1, '["фОльга","алфАвІт"]'), (2, 'null'), (3, NULL)`)
	require.NoError(t, err)
	require.NoError(t, raw.Close())

	db, err := sqlite.Open(t.Context(), path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	return db
}

func TestOpen(main *testing.T) {
	main.Run("SeedsEmptyDatabase", func(t *testing.T) {
		words, err := sqlite.NewAccents(openFresh(t)).All(t.Context())
		require.NoError(t, err)
		require.Len(t, words, 233)

		catalog, err := accent.NewCatalog(words)
		require.NoError(t, err)
		require.Len(t, catalog.Lookup("фольга"), 1)
	})

	main.Run("LeavesExistingWordsAlone", func(t *testing.T) {
		words, err := sqlite.NewAccents(openLegacy(t)).All(t.Context())
		require.NoError(t, err)
		require.Len(t, words, 5, "an existing word list must not be re-seeded")
	})

	main.Run("IsIdempotent", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "naholosy.db")

		for range 2 {
			db, err := sqlite.Open(t.Context(), path)
			require.NoError(t, err)
			require.NoError(t, db.Close())
		}

		db, err := sqlite.Open(t.Context(), path)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, db.Close()) })

		words, err := sqlite.NewAccents(db).All(t.Context())
		require.NoError(t, err)
		require.Len(t, words, 233, "reopening must not seed the words again")
	})
}

func TestUsers(main *testing.T) {
	main.Run("ReadsLegacyRows", func(t *testing.T) {
		users := sqlite.NewUsers(openLegacy(t))

		u, err := users.Get(t.Context(), 1)
		require.NoError(t, err)
		require.Equal(t, user.StageGame, u.Stage)
		require.True(t, u.TutorialSeen)
	})

	main.Run("TreatsNullsAsDefaults", func(t *testing.T) {
		u, err := sqlite.NewUsers(openLegacy(t)).Get(t.Context(), 2)
		require.NoError(t, err)
		require.Equal(t, user.StageStart, u.Stage)
		require.False(t, u.TutorialSeen)
	})

	main.Run("EnsureCreatesOnce", func(t *testing.T) {
		users := sqlite.NewUsers(openFresh(t))

		created, err := users.Ensure(t.Context(), 42)
		require.NoError(t, err)
		require.Equal(t, user.StageStart, created.Stage)

		created.MoveTo(user.StageMainMenu)
		require.NoError(t, users.Save(t.Context(), created))

		again, err := users.Ensure(t.Context(), 42)
		require.NoError(t, err)
		require.Equal(t, user.StageMainMenu, again.Stage, "Ensure must not reset an existing user")

		ids, err := users.IDs(t.Context())
		require.NoError(t, err)
		require.Equal(t, []int64{42}, ids)
	})

	main.Run("EnsureKeepsLegacyUser", func(t *testing.T) {
		u, err := sqlite.NewUsers(openLegacy(t)).Ensure(t.Context(), 1)
		require.NoError(t, err)
		require.Equal(t, user.StageGame, u.Stage)
		require.True(t, u.TutorialSeen)
	})

	main.Run("GetMissing", func(t *testing.T) {
		_, err := sqlite.NewUsers(openFresh(t)).Get(t.Context(), 999)
		require.ErrorIs(t, err, user.ErrNotFound)
	})

	main.Run("SaveMissing", func(t *testing.T) {
		err := sqlite.NewUsers(openFresh(t)).Save(t.Context(), &user.User{ID: 999})
		require.ErrorIs(t, err, user.ErrNotFound)
	})
}

func TestSessions(main *testing.T) {
	main.Run("RestoresRunInProgress", func(t *testing.T) {
		s, err := sqlite.NewSessions(openLegacy(t)).Load(t.Context(), 1)
		require.NoError(t, err)
		require.Equal(t, 12, s.Size)
		require.Equal(t, 3, s.Correct)
		require.Equal(t, 1, s.Wrong)
		require.Equal(t, "вИгода", s.CurrentWord)
		require.Equal(t, []string{"фОльга"}, s.AskedWords)
	})

	main.Run("NoRunWithoutLength", func(t *testing.T) {
		_, err := sqlite.NewSessions(openLegacy(t)).Load(t.Context(), 2)
		require.ErrorIs(t, err, practice.ErrNoSession)
	})

	main.Run("SurvivesMalformedAskedWords", func(t *testing.T) {
		s, err := sqlite.NewSessions(openLegacy(t)).Load(t.Context(), 3)
		require.NoError(t, err)
		require.Empty(t, s.AskedWords)
	})

	main.Run("RoundTrips", func(t *testing.T) {
		db := openLegacy(t)
		sessions := sqlite.NewSessions(db)

		s, err := practice.NewSession(1, 24)
		require.NoError(t, err)
		s.Ask("фОльга")
		_, _ = s.Answer("фОльга")
		s.Ask("алфАвІт")
		require.NoError(t, sessions.Save(t.Context(), s))

		loaded, err := sessions.Load(t.Context(), 1)
		require.NoError(t, err)
		require.Equal(t, 24, loaded.Size)
		require.Equal(t, 1, loaded.Answered)
		require.Equal(t, 1, loaded.Correct)
		require.Equal(t, "алфАвІт", loaded.CurrentWord)
		require.ElementsMatch(t, []string{"фОльга", "алфАвІт"}, loaded.AskedWords)
	})

	main.Run("ResetClearsCounters", func(t *testing.T) {
		sessions := sqlite.NewSessions(openLegacy(t))

		require.NoError(t, sessions.Reset(t.Context(), 1))

		s, err := sessions.Load(t.Context(), 1)
		require.NoError(t, err)
		require.Zero(t, s.Answered)
		require.Zero(t, s.Correct)
		require.Zero(t, s.Wrong)
		require.Empty(t, s.CurrentWord)
		require.Empty(t, s.AskedWords)
		require.Equal(t, 12, s.Size, "the chosen length survives a reset")
	})
}

func TestLearnedWords(main *testing.T) {
	main.Run("ReadsLegacyBlob", func(t *testing.T) {
		learned, err := sqlite.NewLearnedWords(openLegacy(t)).List(t.Context(), 1)
		require.NoError(t, err)
		require.Equal(t, []string{"фОльга", "алфАвІт"}, learned)
	})

	main.Run("TreatsNullAndMissingAsEmpty", func(t *testing.T) {
		repo := sqlite.NewLearnedWords(openLegacy(t))

		for _, id := range []int64{2, 3, 404} {
			learned, err := repo.List(t.Context(), id)
			require.NoError(t, err)
			require.Empty(t, learned)
		}
	})

	main.Run("AddIsIdempotent", func(t *testing.T) {
		repo := sqlite.NewLearnedWords(openLegacy(t))

		require.NoError(t, repo.Add(t.Context(), 1, "вИгода"))
		require.NoError(t, repo.Add(t.Context(), 1, "вИгода"))

		learned, err := repo.List(t.Context(), 1)
		require.NoError(t, err)
		require.Equal(t, []string{"фОльга", "алфАвІт", "вИгода"}, learned)
	})

	main.Run("Remove", func(t *testing.T) {
		repo := sqlite.NewLearnedWords(openLegacy(t))

		require.NoError(t, repo.Remove(t.Context(), 1, "фОльга"))
		require.NoError(t, repo.Remove(t.Context(), 1, "не було такого"))

		learned, err := repo.List(t.Context(), 1)
		require.NoError(t, err)
		require.Equal(t, []string{"алфАвІт"}, learned)
	})
}

func TestStats(main *testing.T) {
	main.Run("CountsLegacyUsers", func(t *testing.T) {
		data, err := sqlite.NewStats(openLegacy(t)).Collect(t.Context())
		require.NoError(t, err)
		require.Equal(t, 3, data.Users)
	})

	main.Run("HandlesEmptyDatabase", func(t *testing.T) {
		data, err := sqlite.NewStats(openFresh(t)).Collect(t.Context())
		require.NoError(t, err)
		require.Zero(t, data.Users)
	})
}

// TestEveryWordIsAnswerable runs the whole seeded list through the option
// builder. Whatever the random draw of double-stress options, the correct
// spelling must be on the keyboard, and nothing on it may be a different word.
func TestEveryWordIsAnswerable(t *testing.T) {
	words, err := sqlite.NewAccents(openFresh(t)).All(t.Context())
	require.NoError(t, err)
	require.Len(t, words, 233)

	catalog, err := accent.NewCatalog(words)
	require.NoError(t, err)

	longest := 0
	for _, a := range catalog.All() {
		for doubles := range 4 {
			variants, vErr := a.Variants(doubles)
			require.NoErrorf(t, vErr, "building options for %q", a.Word)
			require.Containsf(t, variants, a.Word, "the answer to %q was not on the keyboard", a.Word)

			for _, v := range variants {
				require.Equalf(t, a.Plain(), accent.Accent{Word: v}.Plain(),
					"option %q is not a spelling of %q", v, a.Word)
			}

			longest = max(longest, len(variants))
		}
	}

	require.LessOrEqual(t, longest, 12, "the answer keyboard has grown unwieldy")
}
