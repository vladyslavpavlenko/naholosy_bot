package words_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/words"
)

func TestGetPossibleAccents(main *testing.T) {
	main.Run(
		"OK_OneVowel", func(t *testing.T) {
			accents, err := words.GetPossibleAccents("кіт")
			require.NoError(t, err)
			require.Len(t, accents, 1)
			require.Equal(t, []string{"кІт"}, accents)
		},
	)

	main.Run(
		"OK_ThreeVowels", func(t *testing.T) {
			accents, err := words.GetPossibleAccents("крицевий")
			require.NoError(t, err)
			require.Len(t, accents, 3)
			require.Equal(t, []string{"крИцевий", "крицЕвий", "крицевИй"}, accents)
		},
	)

	main.Run(
		"Error_NotInUkrainian_Latin", func(t *testing.T) {
			accents, err := words.GetPossibleAccents("hello")
			require.EqualErrorf(t, err, "word 'hello' is not in ukrainian", "%s", err)
			require.Nil(t, accents)
		},
	)

	main.Run(
		"Error_NotInUkrainian_Random", func(t *testing.T) {
			accents, err := words.GetPossibleAccents("21#_Rzї")
			require.EqualErrorf(t, err, "word '21#_Rzї' is not in ukrainian", "%s", err)
			require.Nil(t, accents)
		},
	)

	main.Run(
		"Error_Empty", func(t *testing.T) {
			accents, err := words.GetPossibleAccents("")
			require.EqualErrorf(t, err, "word is empty", "%s", err)
			require.Nil(t, accents)
		},
	)
}

func TestParseAccentMask(main *testing.T) {
	main.Run(
		"OK_SingleAccent", func(t *testing.T) {
			mask, err := words.ParseAccentMask("кІт")
			require.NoError(t, err)
			require.Equal(t, words.AccentMask{1}, mask)
		},
	)

	main.Run(
		"OK_DoubleAccent", func(t *testing.T) {
			mask, err := words.ParseAccentMask("алфАвІт")
			require.NoError(t, err)
			require.Equal(t, words.AccentMask{3, 5}, mask)
		},
	)

	main.Run(
		"OK_NoAccents", func(t *testing.T) {
			mask, err := words.ParseAccentMask("алфавіт")
			require.NoError(t, err)
			require.Empty(t, mask)
		},
	)

	main.Run(
		"OK_AllCaps", func(t *testing.T) {
			mask, err := words.ParseAccentMask("СТОЛИЦЯ")
			require.NoError(t, err)
			require.Equal(t, words.AccentMask{2, 4, 6}, mask)
		},
	)

	main.Run(
		"Error_Empty", func(t *testing.T) {
			mask, err := words.ParseAccentMask("")
			require.EqualError(t, err, "word is empty")
			require.Nil(t, mask)
		},
	)

	main.Run(
		"Error_NotInUkrainian", func(t *testing.T) {
			mask, err := words.ParseAccentMask("hello")
			require.EqualError(t, err, "word 'hello' is not in ukrainian")
			require.Nil(t, mask)
		},
	)
}

func TestGenerateAccentVariants(main *testing.T) {
	main.Run(
		"OK_SingleAccent", func(t *testing.T) {
			opts := words.Options{MaxAccents: 1, IncludeCorrect: false}
			variants, err := words.GenerateAccentVariants("коло", words.AccentMask{1}, opts)
			require.NoError(t, err)
			require.Len(t, variants, 1)
			require.Equal(t, []string{"колО"}, variants)
		},
	)

	main.Run(
		"OK_DoubleAccent", func(t *testing.T) {
			opts := words.Options{MaxAccents: 2, IncludeCorrect: false}
			variants, err := words.GenerateAccentVariants("алфавіт", words.AccentMask{3, 5}, opts)
			require.NoError(t, err)
			require.Len(t, variants, 5) // 3 single + 2 double accent combinations
			require.Equal(t, []string{"Алфавіт", "АлфАвіт", "АлфавІт", "алфАвіт", "алфавІт"}, variants)
			require.NotContains(t, variants, "алфАвІт")
		},
	)

	main.Run(
		"OK_IncludeCorrect", func(t *testing.T) {
			opts := words.Options{MaxAccents: 1, IncludeCorrect: true}
			variants, err := words.GenerateAccentVariants("навчання", words.AccentMask{4}, opts)
			require.NoError(t, err)
			require.Len(t, variants, 3)
			require.Equal(t, []string{"нАвчання", "навчАння", "навчаннЯ"}, variants)
			require.Contains(t, variants, "навчАння")
		},
	)

	main.Run(
		"OK_NoDuplicates", func(t *testing.T) {
			opts := words.Options{MaxAccents: 2, IncludeCorrect: true}
			variants, err := words.GenerateAccentVariants("аа", words.AccentMask{0, 1}, opts)
			require.NoError(t, err)
			require.Len(t, variants, 3) // 2 single + 1 double
		},
	)

	main.Run(
		"OK_ResultsInEmpty", func(t *testing.T) {
			opts := words.Options{MaxAccents: 1, IncludeCorrect: false}
			variants, err := words.GenerateAccentVariants("кіт", words.AccentMask{1}, opts)
			require.NoError(t, err)
			require.Empty(t, variants)
		},
	)

	main.Run(
		"Error_Empty", func(t *testing.T) {
			opts := words.Options{MaxAccents: 1, IncludeCorrect: false}
			variants, err := words.GenerateAccentVariants("", words.AccentMask{}, opts)
			require.EqualError(t, err, "word is empty")
			require.Nil(t, variants)
		},
	)

	main.Run(
		"Error_NotInUkrainian", func(t *testing.T) {
			opts := words.Options{MaxAccents: 1, IncludeCorrect: false}
			variants, err := words.GenerateAccentVariants("hello", words.AccentMask{}, opts)
			require.EqualError(t, err, "word 'hello' is not in ukrainian")
			require.Nil(t, variants)
		},
	)

	main.Run(
		"Error_MaxAccents_TooLow", func(t *testing.T) {
			opts := words.Options{MaxAccents: 0, IncludeCorrect: false}
			variants, err := words.GenerateAccentVariants("кіт", words.AccentMask{}, opts)
			require.EqualError(t, err, "max accents must be between 1 and 2")
			require.Nil(t, variants)
		},
	)

	main.Run(
		"Error_MaxAccents_TooHigh", func(t *testing.T) {
			opts := words.Options{MaxAccents: 3, IncludeCorrect: false}
			variants, err := words.GenerateAccentVariants("кіт", words.AccentMask{}, opts)
			require.EqualError(t, err, "max accents must be between 1 and 2")
			require.Nil(t, variants)
		},
	)
}
