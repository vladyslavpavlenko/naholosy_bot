package responses_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/accent"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/practice"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/responses"
)

// TestTextsContainRealNewlines guards against the multi-line texts being
// written as raw string literals with a literal backslash-n in them.
func TestTextsContainRealNewlines(t *testing.T) {
	for name, text := range map[string]string{
		"Start":           responses.Start,
		"MainMenu":        responses.MainMenu,
		"WordsMenu":       responses.WordsMenu,
		"PracticeMenu":    responses.PracticeMenu,
		"ResultsTimedOut": responses.ResultsTimedOut,
	} {
		require.NotContains(t, text, `\n`, "%s must use real newlines", name)
		require.Contains(t, text, "\n", "%s is meant to span several lines", name)
	}
}

func TestQuizQuestion(main *testing.T) {
	main.Run("WithNote", func(t *testing.T) {
		text := responses.QuizQuestion(practice.Question{
			Number: 3, Total: 12, Word: "вигода", Note: "(користь)",
		})
		require.Equal(t, "вигода\n(користь)", text)
	})

	main.Run("CarriesNoCounter", func(t *testing.T) {
		// The counter lives in the poll's description, so that the question
		// itself is nothing but the word.
		text := responses.QuizQuestion(practice.Question{Number: 3, Total: 12, Word: "вигода"})
		require.NotContains(t, text, "3")
		require.NotContains(t, text, "/")
	})

	main.Run("CarriesNoMarkup", func(t *testing.T) {
		// A poll question takes only custom emoji, so any tag would be shown
		// to the user verbatim.
		text := responses.QuizQuestion(practice.Question{
			Number: 1, Total: 12, Word: "фольга", Note: "(нотатка)",
		})
		require.NotContains(t, text, "<")
		require.NotContains(t, text, "&")
	})

	main.Run("FitsTelegramsLimit", func(t *testing.T) {
		// A poll question is capped at 300 characters.
		text := responses.QuizQuestion(practice.Question{
			Number: 36, Total: 36, Word: "сільськогосподарський", Note: "(повідомлення, дані, популярність)",
		})
		require.LessOrEqual(t, len([]rune(text)), 300)
	})

	main.Run("WithoutNote", func(t *testing.T) {
		text := responses.QuizQuestion(practice.Question{Number: 1, Total: 12, Word: "фольга"})
		require.Equal(t, "фольга", text)
	})
}

func TestQuizProgress(main *testing.T) {
	main.Run("CountsTheQuestion", func(t *testing.T) {
		require.Equal(t, "3 / 12", responses.QuizProgress(practice.Question{Number: 3, Total: 12}))
	})

	main.Run("FitsTelegramsLimit", func(t *testing.T) {
		// A poll description is capped at 1024 characters.
		text := responses.QuizProgress(practice.Question{Number: 36, Total: 36})
		require.LessOrEqual(t, len([]rune(text)), 1024)
	})
}

func TestQuizExplanation(main *testing.T) {
	main.Run("ShowsTheSpelling", func(t *testing.T) {
		require.Contains(t, responses.QuizExplanation("фОльга", ""), "фОльга")
	})

	main.Run("ShowsTheNoteHeldBackDuringTheQuestion", func(t *testing.T) {
		// "подвійний наголос" is withheld while the question is open, since it
		// names the answer. Once answered it is worth reading.
		text := responses.QuizExplanation("алфАвІт", "(подвійний наголос)")
		require.Contains(t, text, "алфАвІт")
		require.Contains(t, text, "(подвійний наголос)")
	})

	main.Run("FitsTelegramsLimit", func(t *testing.T) {
		// An explanation is capped at 200 characters.
		text := responses.QuizExplanation("сІльськогосподарський", "(повідомлення, дані, популярність)")
		require.LessOrEqual(t, len([]rune(text)), 200)
	})

	main.Run("NoNoteLeavesNoEmptyTag", func(t *testing.T) {
		require.NotContains(t, responses.QuizExplanation("фОльга", ""), "<i></i>")
	})
}

func TestResults(main *testing.T) {
	session := &practice.Session{Size: 12, Answered: 12, Correct: 9, Wrong: 3}

	main.Run("Tally", func(t *testing.T) {
		text := responses.Results(practice.Summary{Session: session})
		require.Contains(t, text, "Тренування завершено")
		require.Contains(t, text, "<b>12 / 12</b>")
		require.Contains(t, text, "<b>9</b>")
		require.Contains(t, text, "<b>3</b>")
	})

	main.Run("WalkingAwaySaysSoInTheSameMessage", func(t *testing.T) {
		// The tally is reported either way; only the heading differs, so that
		// stopping does not take two messages.
		text := responses.Results(practice.Summary{Session: session, TimedOut: true})
		require.Contains(t, text, "Тренування зупинено")
		require.Contains(t, text, "Схоже, тебе немає поруч")
		require.Contains(t, text, "<b>12 / 12</b>")
		require.NotContains(t, text, "Тренування завершено")
	})

	main.Run("NamesTheMissedWords", func(t *testing.T) {
		text := responses.Results(practice.Summary{
			Session: session,
			Missed:  []string{"фОльга", "алфАвІт"},
		})
		require.Contains(t, text, "Варто повторити")
		require.Contains(t, text, "фОльга")
		require.Contains(t, text, "алфАвІт")
	})

	main.Run("SaysNothingWhenThereWereNoMisses", func(t *testing.T) {
		text := responses.Results(practice.Summary{Session: session})
		require.NotContains(t, text, "Варто повторити")
	})
}

func TestWordList(main *testing.T) {
	found := []accent.Accent{{Word: "вИгода", Note: "(користь)"}, {Word: "фОльга"}}

	main.Run("ListsWords", func(t *testing.T) {
		text := responses.WordList([]rune{'в', 'ф'}, found)
		require.Contains(t, text, "В Ф")
		require.Contains(t, text, "вИгода <i>(користь)</i>")
		require.Contains(t, text, "фОльга\n")
		require.NotContains(t, text, "фОльга <i>")
	})

	main.Run("GroupsSeveralLettersUnderHeadings", func(t *testing.T) {
		text := responses.WordList([]rune{'в', 'ф'}, found)
		require.Contains(t, text, "<b>В</b>\nвИгода")
		require.Contains(t, text, "<b>Ф</b>\nфОльга")
	})

	main.Run("NoHeadingForASingleLetter", func(t *testing.T) {
		// The message already says which letter it is.
		text := responses.WordList([]rune{'ф'}, []accent.Accent{{Word: "фОльга"}})
		require.NotContains(t, text, "<b>Ф</b>\n")
	})

	main.Run("SaysSoWhenThereAreNone", func(t *testing.T) {
		text := responses.WordList([]rune{'ю'}, nil)
		require.Contains(t, text, responses.NoWordsForLetters)
	})
}

func TestReaction(main *testing.T) {
	main.Run("EveryGradeHasOne", func(t *testing.T) {
		for _, grade := range []practice.Grade{
			practice.GradePoor, practice.GradeFair, practice.GradeExcellent,
		} {
			require.NotEmpty(t, responses.Reaction(grade))
		}
	})

	main.Run("UnknownGradeDoesNotPanic", func(t *testing.T) {
		require.NotEmpty(t, responses.Reaction(practice.Grade(99)))
		require.NotEmpty(t, responses.Reaction(practice.Grade(-1)))
	})
}
