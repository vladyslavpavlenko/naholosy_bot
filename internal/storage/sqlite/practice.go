package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/vladyslavpavlenko/naholosy_bot/internal/practice"
)

// Sessions stores the run in progress in the user's own row, the way the old
// schema does.
type Sessions struct {
	db *sql.DB
}

// NewSessions wires a practice session repository.
func NewSessions(db *sql.DB) *Sessions {
	return &Sessions{db: db}
}

// Load returns the user's run, or [practice.ErrNoSession] when no length has
// been chosen yet.
func (r *Sessions) Load(ctx context.Context, userID int64) (*practice.Session, error) {
	s := practice.Session{UserID: userID}

	var asked string
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(practice_mode, 0), COALESCE(answered_count, 0), COALESCE(correct_answers, 0),
		       COALESCE(wrong_answers, 0), COALESCE(asked_word, ''), COALESCE(already_asked_words, '')
		FROM users WHERE user_id = ?`, userID,
	).Scan(&s.Size, &s.Answered, &s.Correct, &s.Wrong, &s.CurrentWord, &asked)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, practice.ErrNoSession
	case err != nil:
		return nil, fmt.Errorf("sqlite: loading session of %d: %w", userID, err)
	}

	if !practice.ValidSize(s.Size) {
		return nil, practice.ErrNoSession
	}

	s.AskedWords = decodeWords(asked)

	return &s, nil
}

// Save persists the run.
func (r *Sessions) Save(ctx context.Context, s *practice.Session) error {
	asked, err := json.Marshal(s.AskedWords)
	if err != nil {
		return fmt.Errorf("sqlite: encoding asked words of %d: %w", s.UserID, err)
	}

	if _, err = r.db.ExecContext(ctx, `
		UPDATE users
		SET practice_mode = ?, answered_count = ?, correct_answers = ?, wrong_answers = ?,
		    asked_word = ?, already_asked_words = ?
		WHERE user_id = ?`,
		s.Size, s.Answered, s.Correct, s.Wrong, s.CurrentWord, string(asked), s.UserID,
	); err != nil {
		return fmt.Errorf("sqlite: saving session of %d: %w", s.UserID, err)
	}

	return nil
}

// Reset clears the counters and the asked words, ending any run. The chosen
// length is kept, as the old bot did, so the practice menu still shows it.
func (r *Sessions) Reset(ctx context.Context, userID int64) error {
	if _, err := r.db.ExecContext(ctx, `
		UPDATE users
		SET answered_count = 0, correct_answers = 0, wrong_answers = 0,
		    asked_word = '', already_asked_words = ''
		WHERE user_id = ?`, userID,
	); err != nil {
		return fmt.Errorf("sqlite: resetting session of %d: %w", userID, err)
	}

	return nil
}

// LearnedWords stores learned words as the JSON array the old schema keeps in
// users_xp.
type LearnedWords struct {
	db *sql.DB
}

// NewLearnedWords wires a learned-words repository.
func NewLearnedWords(db *sql.DB) *LearnedWords {
	return &LearnedWords{db: db}
}

// List returns the user's learned words.
func (r *LearnedWords) List(ctx context.Context, userID int64) ([]string, error) {
	var blob string
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(learned_words, '') FROM users_xp WHERE user_id = ?`, userID).Scan(&blob)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("sqlite: listing learned words of %d: %w", userID, err)
	}

	return decodeWords(blob), nil
}

// Add marks a word as learned.
func (r *LearnedWords) Add(ctx context.Context, userID int64, word string) error {
	return r.update(ctx, userID, func(learned []string) []string {
		if slices.Contains(learned, word) {
			return learned
		}

		return append(learned, word)
	})
}

// Remove un-marks a word.
func (r *LearnedWords) Remove(ctx context.Context, userID int64, word string) error {
	return r.update(ctx, userID, func(learned []string) []string {
		return slices.DeleteFunc(learned, func(w string) bool { return w == word })
	})
}

// update rewrites the JSON array. Callers hold the per-user lock and the pool
// has a single connection, so the read and the write cannot interleave.
func (r *LearnedWords) update(ctx context.Context, userID int64, apply func([]string) []string) error {
	learned, err := r.List(ctx, userID)
	if err != nil {
		return err
	}

	blob, err := json.Marshal(apply(learned))
	if err != nil {
		return fmt.Errorf("sqlite: encoding learned words of %d: %w", userID, err)
	}

	if _, err = r.db.ExecContext(ctx,
		`INSERT INTO users_xp (user_id, learned_words) VALUES (?, ?)
		 ON CONFLICT (user_id) DO UPDATE SET learned_words = excluded.learned_words`,
		userID, string(blob),
	); err != nil {
		return fmt.Errorf("sqlite: saving learned words of %d: %w", userID, err)
	}

	return nil
}

// decodeWords reads one of the old bot's JSON string arrays. It wrote absent,
// null and malformed values alike, all of which mean "no words".
func decodeWords(blob string) []string {
	var words []string
	if err := json.Unmarshal([]byte(blob), &words); err != nil {
		return nil
	}

	return words
}
