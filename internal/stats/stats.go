// Package stats answers the operator's question "how is the bot doing?", from
// the durable user data and from this process's counters.
package stats

import "context"

// Data is everything read out of storage.
type Data struct {
	Users int
}

// Repository reads aggregate statistics out of storage.
type Repository interface {
	Collect(ctx context.Context) (Data, error)
}
