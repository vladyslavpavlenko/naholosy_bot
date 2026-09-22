package words

import (
	"strings"
	"unicode"
)

// alphabet defines what counts as a letter in user input.
const alphabet = "абвгґдеєжзиіїйклмнопрстуфхцчшщьюя"

// Normalize folds a user-typed word into a lookup key: trimmed, lowercased,
// apostrophes reduced to a plain one. "тім'янИй", "тім’яний" and "Тім`яний"
// all normalize to "тім'яний".
func Normalize(word string) string {
	return strings.Map(func(r rune) rune {
		if strings.ContainsRune(Apostrophes, r) {
			return '\''
		}
		return unicode.ToLower(r)
	}, strings.TrimSpace(word))
}

// FoldApostrophes reduces every apostrophe variant to a plain one, leaving
// case alone. Use it when comparing accented spellings, where lowercasing
// would throw away the stress.
func FoldApostrophes(word string) string {
	return strings.Map(func(r rune) rune {
		if strings.ContainsRune(Apostrophes, r) {
			return '\''
		}
		return r
	}, word)
}

// IsLetter reports whether r is a letter of the Ukrainian alphabet, in either
// case.
func IsLetter(r rune) bool {
	return strings.ContainsRune(alphabet, unicode.ToLower(r))
}

// IsLetterList reports whether the message is a list of single letters rather
// than a word, i.e. Ukrainian letters separated by whitespace with no two
// letters adjacent.
//
// "є я" and "ф" are letter lists; "фольга" and "ф 9 а" are not.
func IsLetterList(message string) bool {
	previousWasLetter := false
	found := false

	for _, r := range message {
		switch {
		case IsLetter(r):
			if previousWasLetter {
				return false
			}
			previousWasLetter = true
			found = true
		case unicode.IsSpace(r):
			previousWasLetter = false
		default:
			return false
		}
	}

	return found
}

// Letters extracts the distinct lowercase letters from a message, preserving
// the order in which they first appear. Non-letters are ignored.
func Letters(message string) []rune {
	seen := make(map[rune]struct{}, len(message))
	letters := make([]rune, 0, len(message))

	for _, r := range message {
		if !IsLetter(r) {
			continue
		}

		lower := unicode.ToLower(r)
		if _, ok := seen[lower]; ok {
			continue
		}

		seen[lower] = struct{}{}
		letters = append(letters, lower)
	}

	return letters
}

// FirstLetter returns the first Ukrainian letter of a word in lowercase, or
// zero if the word contains none.
func FirstLetter(word string) rune {
	for _, r := range word {
		if IsLetter(r) {
			return unicode.ToLower(r)
		}
	}

	return 0
}
