package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/vladyslavpavlenko/naholosy_bot/internal/accent"
)

// Accents reads the approved word list.
type Accents struct {
	db *sql.DB
}

// NewAccents wires an accent repository.
func NewAccents(db *sql.DB) *Accents {
	return &Accents{db: db}
}

// All returns the word list in alphabetical order.
func (r *Accents) All(ctx context.Context) ([]accent.Accent, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, word, COALESCE(hint, '') FROM naholosy WHERE word IS NOT NULL AND word != '' ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("sqlite: listing words: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var accents []accent.Accent
	for rows.Next() {
		var a accent.Accent
		if err = rows.Scan(&a.ID, &a.Word, &a.Note); err != nil {
			return nil, fmt.Errorf("sqlite: scanning words: %w", err)
		}
		accents = append(accents, a)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: listing words: %w", err)
	}

	return accents, nil
}
