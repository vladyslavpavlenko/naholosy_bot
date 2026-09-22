// Package sqlite persists the bot's state in the naholosy.db file the first
// version of the bot created. The schema is unchanged, so the new binary can
// replace the old one in place.
package sqlite

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"net/url"

	// Pure-Go driver, so the bot builds with CGO_ENABLED=0.
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

//go:embed seed.sql
var seed string

// Open opens the database and makes sure the schema and word list are there.
//
// The pool holds a single connection: SQLite admits one writer at a time, and
// at this bot's load serializing access removes SQLITE_BUSY entirely.
func Open(ctx context.Context, path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn(path))
	if err != nil {
		return nil, fmt.Errorf("sqlite: opening %s: %w", path, err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err = db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("sqlite: connecting to %s: %w", path, err)
	}

	if err = prepare(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

// prepare creates any missing table and seeds the word list if it is empty.
// On an existing database both are no-ops.
func prepare(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("sqlite: applying schema: %w", err)
	}

	var words int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM naholosy`).Scan(&words); err != nil {
		return fmt.Errorf("sqlite: counting words: %w", err)
	}

	if words > 0 {
		return nil
	}

	if _, err := db.ExecContext(ctx, seed); err != nil {
		return fmt.Errorf("sqlite: seeding words: %w", err)
	}

	return nil
}

func dsn(path string) string {
	q := url.Values{}
	q.Add("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", "synchronous(normal)")
	q.Add("_pragma", "busy_timeout(5000)")

	return "file:" + path + "?" + q.Encode()
}
