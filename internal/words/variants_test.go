package words_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/words"
)

func TestAnswerVariants(main *testing.T) {
	main.Run("SingleStress", func(t *testing.T) {
		variants, err := words.AnswerVariants("фОльга", 0)
		require.NoError(t, err)
		require.Equal(t, []string{"фОльга", "фольгА"}, variants)
	})

	main.Run("DoubleStressAddsTheWordItself", func(t *testing.T) {
		variants, err := words.AnswerVariants("алфАвІт", 0)
		require.NoError(t, err)
		require.Equal(t, []string{"Алфавіт", "алфАвіт", "алфавІт", "алфАвІт"}, variants)
	})

	main.Run("Apostrophe", func(t *testing.T) {
		variants, err := words.AnswerVariants("тім'янИй", 0)
		require.NoError(t, err)
		require.Contains(t, variants, "тім'янИй")
	})

	main.Run("Hyphen", func(t *testing.T) {
		variants, err := words.AnswerVariants("де-Юре", 0)
		require.NoError(t, err)
		require.Equal(t, []string{"дЕ-юре", "де-Юре", "де-юрЕ"}, variants)
	})

	main.Run("Error_NotInUkrainian", func(t *testing.T) {
		variants, err := words.AnswerVariants("hello", 0)
		require.Error(t, err)
		require.Nil(t, variants)
	})
}

func TestAnswerVariantsDoubles(main *testing.T) {
	// Every word in the approved list is stressed once, except five that carry
	// a genuine double stress. A double-stress option must therefore not be a
	// tell, which is what the wrong ones are for.
	const (
		singleStressed = "агронОмія"
		doubleStressed = "алфАвІт"
	)

	main.Run("OffersWrongDoublesForASingleStressedWord", func(t *testing.T) {
		for range 30 {
			variants, err := words.AnswerVariants(singleStressed, 2)
			require.NoError(t, err)
			require.Contains(t, variants, singleStressed)

			doubles := doublesOf(variants)
			require.Len(t, doubles, 2)
			require.NotContains(t, doubles, singleStressed)
		}
	})

	main.Run("CountIsTheSameWhicheverWordItIs", func(t *testing.T) {
		// This is the point of the whole thing: the number of double-stress
		// options must not betray whether one of them is the answer.
		for _, count := range []int{1, 2} {
			for range 30 {
				single, err := words.AnswerVariants(singleStressed, count)
				require.NoError(t, err)

				double, err := words.AnswerVariants(doubleStressed, count)
				require.NoError(t, err)

				require.Len(t, doublesOf(single), count)
				require.Len(t, doublesOf(double), count)
				require.Contains(t, doublesOf(double), doubleStressed)
			}
		}
	})

	main.Run("DoubleStressedWordAlwaysKeepsItsAnswer", func(t *testing.T) {
		// Asking for none would otherwise drop the only correct option.
		for range 30 {
			variants, err := words.AnswerVariants(doubleStressed, 0)
			require.NoError(t, err)
			require.Contains(t, variants, doubleStressed)
			require.Len(t, doublesOf(variants), 1)
		}
	})

	main.Run("CorrectDoubleMovesAround", func(t *testing.T) {
		// "алфАвІт" stresses its last two vowels, so ordering the doubles by
		// position would pin the answer to the last slot every time.
		positions := map[int]int{}
		for range 60 {
			variants, err := words.AnswerVariants(doubleStressed, 2)
			require.NoError(t, err)

			for i, d := range doublesOf(variants) {
				if d == doubleStressed {
					positions[i]++
				}
			}
		}
		require.Greater(t, len(positions), 1, "the correct double always landed in the same slot")
	})

	main.Run("WrongDoublesVaryBetweenQuestions", func(t *testing.T) {
		seen := map[string]struct{}{}
		for range 60 {
			variants, err := words.AnswerVariants(singleStressed, 1)
			require.NoError(t, err)

			for _, d := range doublesOf(variants) {
				seen[d] = struct{}{}
			}
		}
		require.Greater(t, len(seen), 1, "the same wrong double came up every time")
	})

	main.Run("NeverOffersMoreThanExist", func(t *testing.T) {
		// "фОльга" has two vowels, so there is exactly one double variant.
		variants, err := words.AnswerVariants("фОльга", 5)
		require.NoError(t, err)
		require.Equal(t, []string{"фОльга", "фольгА", "фОльгА"}, variants)
	})

	main.Run("SinglesComeFirstAndAreUnchanged", func(t *testing.T) {
		plain, err := words.AnswerVariants(singleStressed, 0)
		require.NoError(t, err)

		withDoubles, err := words.AnswerVariants(singleStressed, 2)
		require.NoError(t, err)
		require.Equal(t, plain, withDoubles[:len(plain)])
	})

	main.Run("NoneAskedForMeansNone", func(t *testing.T) {
		variants, err := words.AnswerVariants(singleStressed, 0)
		require.NoError(t, err)
		require.Empty(t, doublesOf(variants))
	})

	main.Run("EveryOptionIsTheSameWord", func(t *testing.T) {
		variants, err := words.AnswerVariants(doubleStressed, 2)
		require.NoError(t, err)

		for _, v := range variants {
			require.Equal(t, words.Normalize(doubleStressed), words.Normalize(v))
		}
	})
}

func TestHasDoubleStress(main *testing.T) {
	main.Run("Double", func(t *testing.T) {
		require.True(t, words.HasDoubleStress("алфАвІт"))
	})

	main.Run("Single", func(t *testing.T) {
		require.False(t, words.HasDoubleStress("фОльга"))
	})

	main.Run("None", func(t *testing.T) {
		require.False(t, words.HasDoubleStress("фольга"))
	})
}

func doublesOf(variants []string) []string {
	var doubles []string
	for _, v := range variants {
		if words.HasDoubleStress(v) {
			doubles = append(doubles, v)
		}
	}

	return doubles
}
