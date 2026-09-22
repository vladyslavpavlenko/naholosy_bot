// Package accent holds the approved list of Ukrainian words together with the
// vowel that carries their stress, and the rules for looking words up.
package accent

import (
	"github.com/vladyslavpavlenko/naholosy_bot/internal/words"
)

// Accent is one entry of the word list: the word with its stressed vowel in
// uppercase, plus a note that tells homographs apart ("вИгода (користь)" vs
// "вигОда (зручність)").
type Accent struct {
	ID   int64
	Word string
	Note string
}

// Plain returns the word as a user would type it: lowercase, no stress.
func (a Accent) Plain() string {
	return words.Normalize(a.Word)
}

// Letter returns the first letter of the word, in lowercase.
func (a Accent) Letter() rune {
	return words.FirstLetter(a.Word)
}

// Variants returns the options to offer when asking the user to stress this
// word, including that many double-stress ones. The correct answer is always
// among them.
func (a Accent) Variants(doubles int) ([]string, error) {
	return words.AnswerVariants(a.Word, doubles)
}

// Stresses returns the number of stressed vowels, which is one for all but a
// handful of words such as "алфАвІт".
func (a Accent) Stresses() int {
	mask, err := words.ParseAccentMask(a.Word)
	if err != nil {
		return 0
	}

	return len(mask)
}

// DoubleStressed reports whether the word may be stressed more than one way.
func (a Accent) DoubleStressed() bool {
	return a.Stresses() > 1
}
