// Package metrics collects in-process counters about the running bot. There is
// no scrape endpoint; /status is the only consumer.
package metrics

import (
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics is a set of process-lifetime counters, safe for concurrent use.
type Metrics struct {
	startedAt time.Time

	UpdatesTotal    atomic.Int64
	UpdatesFailed   atomic.Int64
	PanicsRecovered atomic.Int64

	MessagesSent atomic.Int64
	SendFailures atomic.Int64

	SessionsStarted   atomic.Int64
	SessionsCompleted atomic.Int64
	SessionsAbandoned atomic.Int64

	AnswersCorrect atomic.Int64
	AnswersWrong   atomic.Int64

	LookupsHit  atomic.Int64
	LookupsMiss atomic.Int64

	BroadcastsDelivered atomic.Int64
	BroadcastsFailed    atomic.Int64

	mu       sync.Mutex
	handlers map[string]*handlerStat
}

type handlerStat struct {
	calls  int64
	errors int64
	total  time.Duration
	min    time.Duration
	max    time.Duration
}

// New returns a Metrics whose uptime starts now.
func New() *Metrics {
	return &Metrics{
		startedAt: time.Now(),
		handlers:  make(map[string]*handlerStat),
	}
}

// StartedAt returns when the process began collecting.
func (m *Metrics) StartedAt() time.Time { return m.startedAt }

// Uptime returns how long the process has been collecting.
func (m *Metrics) Uptime() time.Duration { return time.Since(m.startedAt) }

// ObserveHandler records one completed dispatch of a named handler.
func (m *Metrics) ObserveHandler(name string, took time.Duration, failed bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	stat, ok := m.handlers[name]
	if !ok {
		stat = &handlerStat{}
		m.handlers[name] = stat
	}

	stat.calls++
	stat.total += took
	if stat.calls == 1 || took < stat.min {
		stat.min = took
	}
	if took > stat.max {
		stat.max = took
	}
	if failed {
		stat.errors++
	}
}

// HandlerSnapshot is the aggregate timing of one handler.
type HandlerSnapshot struct {
	Name    string
	Calls   int64
	Errors  int64
	Average time.Duration
	Min     time.Duration
	Max     time.Duration
}

// Snapshot is a read of every counter. They are read independently, so under
// load a snapshot may mix adjacent instants.
type Snapshot struct {
	Uptime time.Duration

	UpdatesTotal    int64
	UpdatesFailed   int64
	PanicsRecovered int64

	MessagesSent int64
	SendFailures int64

	SessionsStarted   int64
	SessionsCompleted int64
	SessionsAbandoned int64

	AnswersCorrect int64
	AnswersWrong   int64

	LookupsHit  int64
	LookupsMiss int64

	BroadcastsDelivered int64
	BroadcastsFailed    int64

	// Handlers is ordered by descending call count.
	Handlers []HandlerSnapshot
}

// Snapshot reads every counter.
func (m *Metrics) Snapshot() Snapshot {
	s := Snapshot{
		Uptime:              m.Uptime(),
		UpdatesTotal:        m.UpdatesTotal.Load(),
		UpdatesFailed:       m.UpdatesFailed.Load(),
		PanicsRecovered:     m.PanicsRecovered.Load(),
		MessagesSent:        m.MessagesSent.Load(),
		SendFailures:        m.SendFailures.Load(),
		SessionsStarted:     m.SessionsStarted.Load(),
		SessionsCompleted:   m.SessionsCompleted.Load(),
		SessionsAbandoned:   m.SessionsAbandoned.Load(),
		AnswersCorrect:      m.AnswersCorrect.Load(),
		AnswersWrong:        m.AnswersWrong.Load(),
		LookupsHit:          m.LookupsHit.Load(),
		LookupsMiss:         m.LookupsMiss.Load(),
		BroadcastsDelivered: m.BroadcastsDelivered.Load(),
		BroadcastsFailed:    m.BroadcastsFailed.Load(),
	}

	m.mu.Lock()
	s.Handlers = make([]HandlerSnapshot, 0, len(m.handlers))
	for name, stat := range m.handlers {
		avg := time.Duration(0)
		if stat.calls > 0 {
			avg = stat.total / time.Duration(stat.calls)
		}
		s.Handlers = append(s.Handlers, HandlerSnapshot{
			Name:    name,
			Calls:   stat.calls,
			Errors:  stat.errors,
			Average: avg,
			Min:     stat.min,
			Max:     stat.max,
		})
	}
	m.mu.Unlock()

	sort.Slice(s.Handlers, func(i, j int) bool {
		if s.Handlers[i].Calls != s.Handlers[j].Calls {
			return s.Handlers[i].Calls > s.Handlers[j].Calls
		}
		return s.Handlers[i].Name < s.Handlers[j].Name
	})

	return s
}
