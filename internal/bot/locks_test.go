package bot_test

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/bot"
)

// TestKeyedMutexSerialisesPerKey checks that work for one key never overlaps,
// which is what keeps a user's read-modify-write state consistent, while work
// for different keys is free to run at the same time.
func TestKeyedMutexSerialisesPerKey(t *testing.T) {
	const (
		keys   = 4
		perKey = 50
	)

	// Each key owns its own slot, so anything the race detector reports here
	// is the lock failing to exclude, not the test sharing state.
	type slot struct {
		inside  bool
		entered int
	}

	locks := bot.NewKeyedMutex()
	slots := make([]slot, keys)
	var overlapped atomic.Bool

	var wg sync.WaitGroup
	for k := range keys {
		for range perKey {
			wg.Add(1)
			go func() {
				defer wg.Done()

				release := locks.Lock(int64(k))
				defer release()

				if slots[k].inside {
					overlapped.Store(true)
				}

				slots[k].inside = true
				slots[k].entered++
				slots[k].inside = false
			}()
		}
	}
	wg.Wait()

	require.False(t, overlapped.Load(), "two goroutines held the same key at once")
	for k := range keys {
		require.Equal(t, perKey, slots[k].entered)
	}
}
