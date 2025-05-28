// Package words provides the functionality needed to work with words.
package words

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

const vowels = "аеєиіїоуюя"

var ukrainian = regexp.MustCompile(`^[А-ЩЬЮЯҐЄІЇа-щьюяґєії]+$`)

// GetPossibleAccents returns a slice of all the possible ways to accent the word
// provided. Accent is denoted as an uppercase letter.
//
// Example: "крицевий" will result into "крИцевий", "крицЕвий", and "крицевИй".
func GetPossibleAccents(word string) ([]string, error) {
	if err := validate(word); err != nil {
		return nil, err
	}

	runes := []rune(strings.ToLower(word))
	var accents []string

	for i, r := range runes {
		if isVowel(r) {
			tmp := make([]rune, len(runes))
			copy(tmp, runes)
			tmp[i] = unicode.ToUpper(r)
			accents = append(accents, string(tmp))
		}
	}
	return accents, nil
}

func validate(word string) error {
	if word == "" {
		return errors.New("word is empty")
	}
	if !ukrainian.MatchString(word) {
		return fmt.Errorf("word '%s' is not in Ukrainian", word)
	}
	return nil
}

func isVowel(letter rune) bool {
	return strings.ContainsRune(vowels, letter)
}
