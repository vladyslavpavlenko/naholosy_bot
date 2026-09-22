package accent

import (
	"fmt"
	"math/rand/v2"

	"github.com/vladyslavpavlenko/naholosy_bot/internal/words"
)

// Catalog is an immutable in-memory index over the word list, built once at
// startup. Keeping it in memory sidesteps SQLite's ASCII-only case folding,
// which cannot match Cyrillic. It is safe for concurrent use.
type Catalog struct {
	accents    []Accent
	byPlain    map[string][]Accent
	byAccented map[string]Accent
	byLetter   map[rune][]Accent
	letters    []rune
}

// NewCatalog indexes the given accents. Their order is preserved and is the
// order words are listed to users in.
func NewCatalog(accents []Accent) (*Catalog, error) {
	if len(accents) == 0 {
		return nil, fmt.Errorf("accent: catalog is empty")
	}

	c := &Catalog{
		accents:    accents,
		byPlain:    make(map[string][]Accent, len(accents)),
		byAccented: make(map[string]Accent, len(accents)),
		byLetter:   make(map[rune][]Accent),
	}

	for _, a := range accents {
		if _, err := a.Variants(0); err != nil {
			return nil, fmt.Errorf("accent: indexing %q: %w", a.Word, err)
		}

		plain := a.Plain()
		c.byPlain[plain] = append(c.byPlain[plain], a)
		c.byAccented[a.Word] = a

		letter := a.Letter()
		if _, ok := c.byLetter[letter]; !ok {
			c.letters = append(c.letters, letter)
		}
		c.byLetter[letter] = append(c.byLetter[letter], a)
	}

	return c, nil
}

// Len returns the number of words in the catalog.
func (c *Catalog) Len() int { return len(c.accents) }

// All returns every word, in catalog order.
func (c *Catalog) All() []Accent { return c.accents }

// Letters returns the distinct first letters present, in catalog order. The
// alphabet keyboard is built from it, so a letter with no words is never
// offered.
func (c *Catalog) Letters() []rune { return c.letters }

// Lookup returns every entry whose plain spelling matches. Homographs such as
// "вигода" yield more than one.
func (c *Catalog) Lookup(query string) []Accent {
	return c.byPlain[words.Normalize(query)]
}

// ByAccented returns the entry with exactly this accented spelling.
func (c *Catalog) ByAccented(word string) (Accent, bool) {
	a, ok := c.byAccented[word]
	return a, ok
}

// ByLetters returns every word starting with one of the given letters.
func (c *Catalog) ByLetters(letters []rune) []Accent {
	wanted := make(map[rune]struct{}, len(letters))
	for _, l := range letters {
		wanted[l] = struct{}{}
	}

	matched := make([]Accent, 0, len(c.accents))
	for _, a := range c.accents {
		if _, ok := wanted[a.Letter()]; ok {
			matched = append(matched, a)
		}
	}

	return matched
}

// RandomExcept draws uniformly from the whole catalog, skipping exclude.
func (c *Catalog) RandomExcept(exclude map[string]struct{}) (Accent, bool) {
	return pick(c.accents, exclude)
}

// RandomAmong draws uniformly from the given spellings, skipping exclude and
// anything the catalog does not know.
func (c *Catalog) RandomAmong(candidates []string, exclude map[string]struct{}) (Accent, bool) {
	pool := make([]Accent, 0, len(candidates))
	for _, word := range candidates {
		if a, ok := c.byAccented[word]; ok {
			pool = append(pool, a)
		}
	}

	return pick(pool, exclude)
}

// pick builds the eligible set before drawing, so it terminates in one pass
// even when almost everything is excluded.
func pick(pool []Accent, exclude map[string]struct{}) (Accent, bool) {
	eligible := make([]Accent, 0, len(pool))
	for _, a := range pool {
		if _, skip := exclude[a.Word]; !skip {
			eligible = append(eligible, a)
		}
	}

	if len(eligible) == 0 {
		return Accent{}, false
	}

	return eligible[rand.IntN(len(eligible))], true //nolint:gosec // picking a word, not a secret
}
