package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/vladyslavpavlenko/naholosy_bot/internal/stats"
)

// Stats aggregates over the users table.
type Stats struct {
	db *sql.DB
}

// NewStats wires a stats repository.
func NewStats(db *sql.DB) *Stats {
	return &Stats{db: db}
}

// Collect gathers the report's figures.
func (r *Stats) Collect(ctx context.Context) (stats.Data, error) {
	var data stats.Data

	if err := r.db.QueryRowContext(ctx,
		`SELECT count(*) FROM users WHERE user_id IS NOT NULL`,
	).Scan(&data.Users); err != nil {
		return stats.Data{}, fmt.Errorf("sqlite: collecting user stats: %w", err)
	}

	return data, nil
}
