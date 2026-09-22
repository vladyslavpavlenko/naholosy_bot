package bot

import "sync"

// KeyedMutex serializes work per key. Telegram delivers a user's updates
// concurrently, and much of the state is read-modify-write, so one user's
// messages must not be handled at the same time.
type KeyedMutex struct {
	mu    sync.Mutex
	locks map[int64]*entry
}

type entry struct {
	mu   sync.Mutex
	refs int
}

// NewKeyedMutex returns an unlocked keyed mutex.
func NewKeyedMutex() *KeyedMutex {
	return &KeyedMutex{locks: make(map[int64]*entry)}
}

// Lock takes the lock for key and returns the function that releases it.
func (k *KeyedMutex) Lock(key int64) func() {
	k.mu.Lock()
	e, ok := k.locks[key]
	if !ok {
		e = &entry{}
		k.locks[key] = e
	}
	e.refs++
	k.mu.Unlock()

	e.mu.Lock()

	return func() {
		e.mu.Unlock()

		k.mu.Lock()
		e.refs--
		if e.refs == 0 {
			delete(k.locks, key)
		}
		k.mu.Unlock()
	}
}
