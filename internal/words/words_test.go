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
			require.EqualErrorf(t, err, "word 'hello' is not in Ukrainian", "%s", err)
			require.Nil(t, accents)
		},
	)

	main.Run(
		"Error_NotInUkrainian_Random", func(t *testing.T) {
			accents, err := words.GetPossibleAccents("21#_Rzї")
			require.EqualErrorf(t, err, "word '21#_Rzї' is not in Ukrainian", "%s", err)
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
