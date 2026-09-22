// Package keyboard builds the reply keyboards the bot is driven by.
package keyboard

import (
	"strconv"
	"unicode"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/practice"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/responses"
)

// alphabetColumns is how many letters fit on one row without wrapping on a
// phone.
const alphabetColumns = 9

// Faster is the single button shown after the greeting.
func Faster() *telego.ReplyKeyboardMarkup {
	return resize(tu.Keyboard(tu.KeyboardRow(tu.KeyboardButton(responses.FasterButton))))
}

// MainMenu is the main menu.
func MainMenu() *telego.ReplyKeyboardMarkup {
	return resize(tu.Keyboard(
		tu.KeyboardRow(tu.KeyboardButton(responses.PracticeButton)),
		tu.KeyboardRow(
			tu.KeyboardButton(responses.AllWordsButton),
			tu.KeyboardButton(responses.DownloadButton),
		),
	))
}

// PracticeSizes offers the run lengths.
func PracticeSizes() *telego.ReplyKeyboardMarkup {
	sizes := make([]telego.KeyboardButton, 0, len(practice.Sizes))
	for _, size := range practice.Sizes {
		sizes = append(sizes, tu.KeyboardButton(strconv.Itoa(size)))
	}

	return resize(tu.Keyboard(sizes, tu.KeyboardRow(tu.KeyboardButton(responses.MenuButton))))
}

// Readiness asks the user to confirm the run is about to start.
func Readiness() *telego.ReplyKeyboardMarkup {
	return resize(tu.Keyboard(
		tu.KeyboardRow(tu.KeyboardButton(responses.YesButton)),
		tu.KeyboardRow(tu.KeyboardButton(responses.BackButton)),
	))
}

// Alphabet offers the letters words are browsed by.
func Alphabet(letters []rune) *telego.ReplyKeyboardMarkup {
	buttons := make([]telego.KeyboardButton, 0, len(letters))
	for _, r := range letters {
		buttons = append(buttons, tu.KeyboardButton(string(unicode.ToUpper(r))))
	}

	rows := tu.KeyboardCols(alphabetColumns, buttons...)
	rows = append(rows, tu.KeyboardRow(tu.KeyboardButton(responses.MenuButton)))

	return resize(tu.KeyboardGrid(rows))
}

// FinishGame is the standing keyboard during a run: the answers are the quiz
// poll's own options, so this is all the keyboard has to carry.
func FinishGame() *telego.ReplyKeyboardMarkup {
	return resize(tu.Keyboard(tu.KeyboardRow(tu.KeyboardButton(responses.FinishGameButton))))
}

// Remove hides the keyboard.
func Remove() *telego.ReplyKeyboardRemove {
	return tu.ReplyKeyboardRemove()
}

func resize(k *telego.ReplyKeyboardMarkup) *telego.ReplyKeyboardMarkup {
	return k.WithResizeKeyboard()
}
