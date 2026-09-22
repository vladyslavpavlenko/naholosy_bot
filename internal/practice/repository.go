package practice

import (
	"context"
	"errors"
)

// ErrNoSession is returned when a user has no run in progress.
var ErrNoSession = errors.New("practice: no active session")

// Repository stores the run a user is in the middle of.
type Repository interface {
	// Load returns the user's run, or [ErrNoSession].
	Load(ctx context.Context, userID int64) (*Session, error)
	// Save persists the run.
	Save(ctx context.Context, s *Session) error
	// Reset clears the counters and the asked words, ending any run.
	Reset(ctx context.Context, userID int64) error
}

// LearnedWords stores the words a user has got right and not missed since.
type LearnedWords interface {
	List(ctx context.Context, userID int64) ([]string, error)
	Add(ctx context.Context, userID int64, word string) error
	Remove(ctx context.Context, userID int64, word string) error
}
