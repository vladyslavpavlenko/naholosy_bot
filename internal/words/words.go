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
	if word == "" {
		return nil, errors.New("word is empty")
	}

	if !ukrainian.MatchString(word) {
		return nil, fmt.Errorf("word '%s' is not in Ukrainian", word)
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

func isVowel(letter rune) bool {
	return strings.ContainsRune(vowels, letter)
}
