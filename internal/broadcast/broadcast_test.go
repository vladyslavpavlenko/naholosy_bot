package broadcast_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/broadcast"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/metrics"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/user"
	"github.com/vladyslavpavlenko/naholosy_bot/pkg/logger"
)

type delivery struct {
	userID int64
	text   string
	copied bool
}

// fakeSender records deliveries and fails for the users in failFor.
type fakeSender struct {
	mu         sync.Mutex
	deliveries []delivery
	failFor    map[int64]bool
}

func (f *fakeSender) SendText(_ context.Context, userID int64, text string) error {
	return f.record(delivery{userID: userID, text: text})
}

func (f *fakeSender) CopyMessage(_ context.Context, userID, _ int64, _ int) error {
	return f.record(delivery{userID: userID, copied: true})
}

func (f *fakeSender) record(d delivery) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.failFor[d.userID] {
		return errors.New("blocked")
	}
	f.deliveries = append(f.deliveries, d)

	return nil
}

type fakeUsers struct{ ids []int64 }

func (f *fakeUsers) Ensure(context.Context, int64) (*user.User, error) { return nil, nil }
func (f *fakeUsers) Get(context.Context, int64) (*user.User, error)    { return nil, user.ErrNotFound }
func (f *fakeUsers) Save(context.Context, *user.User) error            { return nil }
func (f *fakeUsers) IDs(context.Context) ([]int64, error)              { return f.ids, nil }

func newService(t *testing.T, sender *fakeSender, ids ...int64) *broadcast.Service {
	t.Helper()

	return broadcast.NewService(&fakeUsers{ids: ids}, sender, metrics.New(), logger.New("PROD"))
}

func TestPrepareAndTake(main *testing.T) {
	main.Run("RoundTrips", func(t *testing.T) {
		s := newService(t, &fakeSender{})
		s.Prepare(1, broadcast.Payload{Text: "привіт"})

		p, err := s.Take(1)
		require.NoError(t, err)
		require.Equal(t, "привіт", p.Text)
	})

	main.Run("TakeClears", func(t *testing.T) {
		s := newService(t, &fakeSender{})
		s.Prepare(1, broadcast.Payload{Text: "привіт"})

		_, err := s.Take(1)
		require.NoError(t, err)

		_, err = s.Take(1)
		require.ErrorIs(t, err, broadcast.ErrNothingPending)
	})

	main.Run("IsPerAdmin", func(t *testing.T) {
		s := newService(t, &fakeSender{})
		s.Prepare(1, broadcast.Payload{Text: "перший"})
		s.Prepare(2, broadcast.Payload{Text: "другий"})

		p, err := s.Take(2)
		require.NoError(t, err)
		require.Equal(t, "другий", p.Text)

		p, err = s.Take(1)
		require.NoError(t, err)
		require.Equal(t, "перший", p.Text)
	})

	main.Run("NothingPrepared", func(t *testing.T) {
		_, err := newService(t, &fakeSender{}).Take(1)
		require.ErrorIs(t, err, broadcast.ErrNothingPending)
	})
}

func TestDeliver(main *testing.T) {
	main.Run("Text", func(t *testing.T) {
		sender := &fakeSender{}
		s := newService(t, sender, 1, 2, 3)

		report, err := s.Deliver(t.Context(), broadcast.Payload{Text: "<b>привіт</b>"}, []int64{1, 2, 3})
		require.NoError(t, err)
		require.Equal(t, 3, report.Delivered)
		require.Zero(t, report.Failed)
		require.Len(t, sender.deliveries, 3)
		require.Equal(t, "<b>привіт</b>", sender.deliveries[0].text)
	})

	main.Run("CopyPreservesTheOriginal", func(t *testing.T) {
		sender := &fakeSender{}
		s := newService(t, sender, 1)

		_, err := s.Deliver(t.Context(), broadcast.Payload{FromChatID: 7, MessageID: 42}, []int64{1})
		require.NoError(t, err)
		require.True(t, sender.deliveries[0].copied)
	})

	main.Run("OneFailureDoesNotStopTheRest", func(t *testing.T) {
		sender := &fakeSender{failFor: map[int64]bool{2: true}}
		s := newService(t, sender, 1, 2, 3)

		report, err := s.Deliver(t.Context(), broadcast.Payload{Text: "привіт"}, []int64{1, 2, 3})
		require.NoError(t, err)
		require.Equal(t, 2, report.Delivered)
		require.Equal(t, 1, report.Failed)
		require.Equal(t, 3, report.Total)
	})

	main.Run("EmptyPayloadFails", func(t *testing.T) {
		sender := &fakeSender{}
		s := newService(t, sender, 1)

		report, err := s.Deliver(t.Context(), broadcast.Payload{}, []int64{1})
		require.NoError(t, err)
		require.Equal(t, 1, report.Failed)
		require.Empty(t, sender.deliveries)
	})

	main.Run("StopsWhenCancelled", func(t *testing.T) {
		sender := &fakeSender{}
		s := newService(t, sender, 1, 2, 3)

		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		report, err := s.Deliver(ctx, broadcast.Payload{Text: "привіт"}, []int64{1, 2, 3})
		require.ErrorIs(t, err, context.Canceled)
		require.Equal(t, 1, report.Delivered, "the first delivery goes out before the pace kicks in")
	})

	main.Run("NoAudience", func(t *testing.T) {
		report, err := newService(t, &fakeSender{}).Deliver(t.Context(), broadcast.Payload{Text: "x"}, nil)
		require.NoError(t, err)
		require.Zero(t, report.Total)
	})
}

func TestAudience(t *testing.T) {
	ids, err := newService(t, &fakeSender{}, 1, 2, 3).Audience(t.Context())
	require.NoError(t, err)
	require.Equal(t, []int64{1, 2, 3}, ids)
}

func TestIsCopy(t *testing.T) {
	require.True(t, broadcast.Payload{MessageID: 1}.IsCopy())
	require.False(t, broadcast.Payload{Text: "x"}.IsCopy())
}
