package stats

import (
	"context"
	"runtime"
	"time"

	"github.com/vladyslavpavlenko/naholosy_bot/internal/accent"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/metrics"
)

// Runtime describes the process serving the report.
type Runtime struct {
	Uptime     time.Duration
	Goroutines int
	HeapAlloc  uint64
	GoVersion  string
}

// Report is a complete answer to /status.
type Report struct {
	Data
	Runtime     Runtime
	Metrics     metrics.Snapshot
	CatalogSize int
	GeneratedAt time.Time
}

// Service assembles reports.
type Service struct {
	repo    Repository
	metrics *metrics.Metrics
	catalog *accent.Catalog
}

// NewService wires a stats service.
func NewService(repo Repository, m *metrics.Metrics, catalog *accent.Catalog) *Service {
	return &Service{repo: repo, metrics: m, catalog: catalog}
}

// Report gathers everything worth reporting.
func (s *Service) Report(ctx context.Context) (Report, error) {
	data, err := s.repo.Collect(ctx)
	if err != nil {
		return Report{}, err
	}

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	return Report{
		Data: data,
		Runtime: Runtime{
			Uptime:     s.metrics.Uptime(),
			Goroutines: runtime.NumGoroutine(),
			HeapAlloc:  mem.HeapAlloc,
			GoVersion:  runtime.Version(),
		},
		Metrics:     s.metrics.Snapshot(),
		CatalogSize: s.catalog.Len(),
		GeneratedAt: time.Now().UTC(),
	}, nil
}
