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

type AccentMask []int

type Options struct {
	// MaxAccents is the maximum number of accents the word can have (1-2).
	MaxAccents int

	// IncludeCorrect specifies whether the correct variant should be included.
	IncludeCorrect bool
}

// ParseAccentMask extracts accent positions from a word like "алфАвІт".
func ParseAccentMask(word string) (AccentMask, error) {
	if err := validate(word); err != nil {
		return nil, err
	}

	var mask AccentMask
	for i, r := range []rune(word) {
		if isVowel(unicode.ToLower(r)) && unicode.IsUpper(r) {
			mask = append(mask, i)
		}
	}

	return mask, nil
}

// GenerateAccentVariants generates accent variants including incorrect ones.
func GenerateAccentVariants(
	word string,
	correct AccentMask,
	opts Options,
) ([]string, error) {
	if err := validate(word); err != nil {
		return nil, err
	}

	if opts.MaxAccents < 1 || opts.MaxAccents > 2 {
		return nil, errors.New("max accents must be between 1 and 2")
	}

	runes := []rune(strings.ToLower(word))

	// collect vowel positions
	var wordVowels []int
	for i, r := range runes {
		if isVowel(r) {
			wordVowels = append(wordVowels, i)
		}
	}

	var result []string
	seen := make(map[string]struct{})

	// generate combinations
	var dfs func(start int, mask AccentMask)
	dfs = func(start int, mask AccentMask) {
		if len(mask) > 0 && len(mask) <= opts.MaxAccents {
			word := applyAccentMask(word, mask)
			if _, ok := seen[word]; !ok {
				seen[word] = struct{}{}
				result = append(result, word)
			}
		}

		for i := start; i < len(wordVowels); i++ {
			dfs(i+1, append(mask, wordVowels[i]))
		}
	}

	dfs(0, nil)

	// filter out correct variant if IncludeCorrect is false
	if !opts.IncludeCorrect {
		correctWord := applyAccentMask(word, correct) // nolint:errcheck // validated
		filtered := make([]string, 0, len(result))
		for _, variant := range result {
			if variant != correctWord {
				filtered = append(filtered, variant)
			}
		}
		result = filtered
	} else {
		// optionally ensure the correct variant exists
		correctWord := applyAccentMask(word, correct) // nolint:errcheck // validated
		if _, ok := seen[correctWord]; !ok {
			result = append(result, correctWord)
		}
	}

	return result, nil
}

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

// applyAccentMask applies accent mask to a lowercase word.
func applyAccentMask(word string, mask AccentMask) string {
	runes := []rune(strings.ToLower(word))
	for _, i := range mask {
		if i >= 0 && i < len(runes) {
			runes[i] = unicode.ToUpper(runes[i])
		}
	}
	return string(runes)
}

func isVowel(letter rune) bool {
	return strings.ContainsRune(vowels, letter)
}

func validate(word string) error {
	if word == "" {
		return errors.New("word is empty")
	}

	if !ukrainian.MatchString(word) {
		return fmt.Errorf("word '%s' is not in ukrainian", word)
	}

	return nil
}
