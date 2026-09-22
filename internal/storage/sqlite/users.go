package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/vladyslavpavlenko/naholosy_bot/internal/user"
)

// Users stores users in the legacy users table, whose columns hold both the
// conversational state and the current practice run.
type Users struct {
	db *sql.DB
}

// NewUsers wires a user repository.
func NewUsers(db *sql.DB) *Users {
	return &Users{db: db}
}

// Ensure returns the user, creating them on first contact. The users_xp row is
// created alongside, since the old schema keeps them in step.
//
// A new row is filled out in full, combo_count included, so that it looks
// exactly like one the first version of the bot would have written. Nothing
// reads that column any more.
func (r *Users) Ensure(ctx context.Context, id int64) (*user.User, error) {
	if _, err := r.db.ExecContext(ctx,
		`INSERT INTO users (user_id, menu_stage, skip_tutorial, practice_mode,
			already_asked_words, asked_word, correct_answers, wrong_answers, answered_count, combo_count)
		 VALUES (?, ?, 0, 0, '', '', 0, 0, 0, 0)
		 ON CONFLICT (user_id) DO NOTHING`,
		id, string(user.StageStart),
	); err != nil {
		return nil, fmt.Errorf("sqlite: creating user %d: %w", id, err)
	}

	if _, err := r.db.ExecContext(ctx,
		`INSERT INTO users_xp (user_id, learned_words) VALUES (?, '') ON CONFLICT (user_id) DO NOTHING`, id,
	); err != nil {
		return nil, fmt.Errorf("sqlite: creating user %d progress: %w", id, err)
	}

	return r.Get(ctx, id)
}

// Get returns a user, or [user.ErrNotFound].
func (r *Users) Get(ctx context.Context, id int64) (*user.User, error) {
	var (
		u            = user.User{ID: id}
		stage        string
		skipTutorial int
	)

	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(menu_stage, ''), COALESCE(skip_tutorial, 0) FROM users WHERE user_id = ?`, id,
	).Scan(&stage, &skipTutorial)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, user.ErrNotFound
	case err != nil:
		return nil, fmt.Errorf("sqlite: getting user %d: %w", id, err)
	}

	u.Stage = user.ParseStage(stage)
	u.TutorialSeen = skipTutorial != 0

	return &u, nil
}

// Save persists the user's stage and tutorial flag.
func (r *Users) Save(ctx context.Context, u *user.User) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE users SET menu_stage = ?, skip_tutorial = ? WHERE user_id = ?`,
		string(u.Stage), boolToInt(u.TutorialSeen), u.ID,
	)
	if err != nil {
		return fmt.Errorf("sqlite: saving user %d: %w", u.ID, err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("sqlite: saving user %d: %w", u.ID, err)
	}

	if affected == 0 {
		return user.ErrNotFound
	}

	return nil
}

// IDs returns every known user.
func (r *Users) IDs(ctx context.Context) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT user_id FROM users WHERE user_id IS NOT NULL ORDER BY rowid`)
	if err != nil {
		return nil, fmt.Errorf("sqlite: listing users: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var ids []int64
	for rows.Next() {
		var id int64
		if err = rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("sqlite: scanning users: %w", err)
		}
		ids = append(ids, id)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: listing users: %w", err)
	}

	return ids, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}

	return 0
}
