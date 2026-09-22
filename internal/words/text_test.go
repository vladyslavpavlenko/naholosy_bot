package words_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/words"
)

func TestNormalize(main *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "Lowercases", in: "фОльга", want: "фольга"},
		{name: "Trims", in: "  фольга\n", want: "фольга"},
		{name: "TypographicApostrophe", in: "тім’янИй", want: "тім'яний"},
		{name: "ModifierApostrophe", in: "тімʼяний", want: "тім'яний"},
		{name: "BacktickApostrophe", in: "тім`яний", want: "тім'яний"},
		{name: "PlainApostropheUnchanged", in: "тім'яний", want: "тім'яний"},
		{name: "KeepsHyphen", in: "де-Юре", want: "де-юре"},
		{name: "Empty", in: "", want: ""},
	}

	for _, tt := range tests {
		main.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, words.Normalize(tt.in))
		})
	}
}

func TestIsLetterList(main *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{name: "SingleLetter", in: "ф", want: true},
		{name: "UppercaseLetter", in: "Є", want: true},
		{name: "SpacedLetters", in: "ф и в р а", want: true},
		{name: "LeadingAndTrailingSpaces", in: "  є  я  ", want: true},
		{name: "Word", in: "фольга", want: false},
		{name: "LettersAndDigits", in: "ф 9 % а", want: false},
		{name: "OnlySpaces", in: "     ", want: false},
		{name: "Empty", in: "", want: false},
		{name: "Latin", in: "a b", want: false},
		{name: "Emoji", in: "🎯 Практика", want: false},
	}

	for _, tt := range tests {
		main.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, words.IsLetterList(tt.in))
		})
	}
}

func TestLetters(main *testing.T) {
	main.Run("PreservesOrder", func(t *testing.T) {
		require.Equal(t, []rune{'є', 'я'}, words.Letters("Є Я"))
	})

	main.Run("Deduplicates", func(t *testing.T) {
		require.Equal(t, []rune{'а', 'б'}, words.Letters("а Б а б"))
	})

	main.Run("IgnoresNonLetters", func(t *testing.T) {
		require.Empty(t, words.Letters("1 2 %"))
	})
}

func TestFirstLetter(main *testing.T) {
	main.Run("Accented", func(t *testing.T) {
		require.Equal(t, 'а', words.FirstLetter("Аркушик"))
	})

	main.Run("None", func(t *testing.T) {
		require.Equal(t, rune(0), words.FirstLetter("123"))
	})
}
