package user

import (
	"context"
	"errors"
)

// ErrNotFound is returned when no user exists for an ID.
var ErrNotFound = errors.New("user: not found")

// Repository stores users.
type Repository interface {
	// Ensure returns the user, creating them on first contact.
	Ensure(ctx context.Context, id int64) (*User, error)
	// Get returns a user, or [ErrNotFound].
	Get(ctx context.Context, id int64) (*User, error)
	// Save persists the user's stage and tutorial flag.
	Save(ctx context.Context, u *User) error
	// IDs returns every known user, for broadcasting.
	IDs(ctx context.Context) ([]int64, error)
}
