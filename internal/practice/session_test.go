package practice_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/practice"
)

func TestNewSession(main *testing.T) {
	main.Run("AcceptsOfferedLengths", func(t *testing.T) {
		for _, size := range practice.Sizes {
			s, err := practice.NewSession(1, size)
			require.NoError(t, err)
			require.Equal(t, size, s.Size)
		}
	})

	main.Run("RejectsOthers", func(t *testing.T) {
		for _, size := range []int{0, -1, 13, 100} {
			_, err := practice.NewSession(1, size)
			require.Error(t, err)
		}
	})
}

func TestAsk(main *testing.T) {
	main.Run("RecordsTheWord", func(t *testing.T) {
		s := newSession(t, 12)
		s.Ask("фОльга")

		require.Equal(t, "фОльга", s.CurrentWord)
		require.Equal(t, []string{"фОльга"}, s.AskedWords)
		require.Contains(t, s.Asked(), "фОльга")
	})

	main.Run("DoesNotDuplicate", func(t *testing.T) {
		s := newSession(t, 12)
		s.Ask("фОльга")
		s.Ask("фОльга")

		require.Len(t, s.AskedWords, 1)
	})
}

func TestAnswer(main *testing.T) {
	main.Run("Correct", func(t *testing.T) {
		s := newSession(t, 12)
		s.Ask("фОльга")

		correct, ok := s.Answer("фОльга")
		require.True(t, ok)
		require.True(t, correct)
		require.Equal(t, 1, s.Answered)
		require.Equal(t, 1, s.Correct)
		require.Zero(t, s.Wrong)
		require.Empty(t, s.CurrentWord)
	})

	main.Run("Wrong", func(t *testing.T) {
		s := newSession(t, 12)
		s.Ask("фОльга")

		correct, ok := s.Answer("фольгА")
		require.True(t, ok)
		require.False(t, correct)
		require.Equal(t, 1, s.Answered)
		require.Equal(t, 1, s.Wrong)
	})

	main.Run("AWrongWordCanComeBack", func(t *testing.T) {
		s := newSession(t, 12)
		s.Ask("фОльга")
		_, _ = s.Answer("фольгА")

		require.NotContains(t, s.Asked(), "фОльга", "a missed word must stay drawable")
		require.NotContains(t, s.AskedWords, "фОльга")
	})

	main.Run("ARightWordDoesNot", func(t *testing.T) {
		s := newSession(t, 12)
		s.Ask("фОльга")
		_, _ = s.Answer("фОльга")

		require.Contains(t, s.Asked(), "фОльга")
	})

	main.Run("ComingBackDoesNotDisturbTheOthers", func(t *testing.T) {
		s := newSession(t, 12)
		s.Ask("алфАвІт")
		_, _ = s.Answer("алфАвІт")
		s.Ask("фОльга")
		_, _ = s.Answer("фольгА")
		s.Ask("вИгода")
		_, _ = s.Answer("вИгода")

		require.Equal(t, []string{"алфАвІт", "вИгода"}, s.AskedWords)
	})

	main.Run("IgnoresReplyWithNoQuestion", func(t *testing.T) {
		s := newSession(t, 12)

		_, ok := s.Answer("фОльга")
		require.False(t, ok)
		require.Zero(t, s.Answered)
	})

	main.Run("IgnoresDuplicateReply", func(t *testing.T) {
		s := newSession(t, 12)
		s.Ask("фОльга")

		_, ok := s.Answer("фОльга")
		require.True(t, ok)

		_, ok = s.Answer("фОльга")
		require.False(t, ok, "a second reply to the same question must not count twice")
		require.Equal(t, 1, s.Answered)
	})
}

func TestQuestionNumber(t *testing.T) {
	s := newSession(t, 12)
	require.Equal(t, 1, s.QuestionNumber())

	s.Ask("фОльга")
	require.Equal(t, 1, s.QuestionNumber(), "the number is of the question being asked")

	_, _ = s.Answer("фОльга")
	require.Equal(t, 2, s.QuestionNumber())
}

func TestComplete(t *testing.T) {
	s := newSession(t, 12)
	for i := range 12 {
		require.False(t, s.Complete(), "run ended early at answer %d", i)
		s.Ask("фОльга")
		_, _ = s.Answer("фОльга")
	}

	require.True(t, s.Complete())
	require.Equal(t, 12, s.Correct)
}

func TestGrade(main *testing.T) {
	// The thresholds are the original bot's: a third or less is poor, within
	// three of a clean sweep is excellent.
	tests := []struct {
		name    string
		size    int
		correct int
		want    practice.Grade
	}{
		{name: "None", size: 12, correct: 0, want: practice.GradePoor},
		{name: "ExactlyAThird", size: 12, correct: 4, want: practice.GradePoor},
		{name: "JustOverAThird", size: 12, correct: 5, want: practice.GradeFair},
		{name: "JustUnderExcellent", size: 12, correct: 8, want: practice.GradeFair},
		{name: "ThreeOff", size: 12, correct: 9, want: practice.GradeExcellent},
		{name: "CleanSweep", size: 12, correct: 12, want: practice.GradeExcellent},
		{name: "LongRunPoor", size: 36, correct: 12, want: practice.GradePoor},
		{name: "LongRunFair", size: 36, correct: 13, want: practice.GradeFair},
		{name: "LongRunExcellent", size: 36, correct: 33, want: practice.GradeExcellent},
	}

	for _, tt := range tests {
		main.Run(tt.name, func(t *testing.T) {
			s := newSession(t, tt.size)
			s.Correct = tt.correct
			require.Equal(t, tt.want, s.Grade())
		})
	}
}

func newSession(t *testing.T, size int) *practice.Session {
	t.Helper()

	s, err := practice.NewSession(1, size)
	require.NoError(t, err)

	return s
}
