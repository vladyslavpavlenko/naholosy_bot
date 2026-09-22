package keyboard_test

import (
	"testing"

	"github.com/mymmrac/telego"
	"github.com/stretchr/testify/require"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/keyboard"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/practice"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/responses"
)

func TestAlphabet(main *testing.T) {
	// The 27 letters the approved list actually starts with.
	letters := []rune("абвгдеєжзіклмнопрстуфхцчшщя")

	main.Run("NineToARowPlusTheMenu", func(t *testing.T) {
		rows := keyboard.Alphabet(letters).Keyboard
		require.Len(t, rows, 4)
		require.Len(t, rows[0], 9)
		require.Len(t, rows[1], 9)
		require.Len(t, rows[2], 9)
		require.Equal(t, responses.MenuButton, rows[3][0].Text)
	})

	main.Run("Uppercased", func(t *testing.T) {
		require.Equal(t, "А", keyboard.Alphabet(letters).Keyboard[0][0].Text)
	})

	main.Run("OnlyOffersLettersItWasGiven", func(t *testing.T) {
		rows := keyboard.Alphabet([]rune{'ф'}).Keyboard
		require.Len(t, rows, 2)
		require.Equal(t, []string{"Ф"}, texts(rows[0]))
	})
}

func TestPracticeSizes(t *testing.T) {
	rows := keyboard.PracticeSizes().Keyboard
	require.Len(t, rows, 2)
	require.Len(t, rows[0], len(practice.Sizes))
	require.Equal(t, []string{"12", "24", "36"}, texts(rows[0]))
	require.Equal(t, responses.MenuButton, rows[1][0].Text)
}

func TestMainMenu(t *testing.T) {
	rows := keyboard.MainMenu().Keyboard
	require.Equal(t, []string{responses.PracticeButton}, texts(rows[0]))
	require.Equal(t, []string{responses.AllWordsButton, responses.DownloadButton}, texts(rows[1]))
}

func TestKeyboardsResize(t *testing.T) {
	for name, k := range map[string]*telego.ReplyKeyboardMarkup{
		"Faster":        keyboard.Faster(),
		"MainMenu":      keyboard.MainMenu(),
		"PracticeSizes": keyboard.PracticeSizes(),
		"Readiness":     keyboard.Readiness(),
		"FinishGame":    keyboard.FinishGame(),
	} {
		require.True(t, k.ResizeKeyboard, "%s should resize to fit", name)
	}
}

func texts(row []telego.KeyboardButton) []string {
	out := make([]string, 0, len(row))
	for _, b := range row {
		out = append(out, b.Text)
	}

	return out
}
