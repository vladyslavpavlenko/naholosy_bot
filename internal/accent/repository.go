package accent

import "context"

// Repository loads the word list. It is read once at startup to build a
// [Catalog].
type Repository interface {
	// All returns every word in the approved list, ordered alphabetically.
	All(ctx context.Context) ([]Accent, error)
}
