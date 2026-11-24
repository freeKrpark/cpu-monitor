package agent

import (
	"context"
	"cpu-monitor/internal/metric"
	"errors"
	"fmt"
	"log"
	"time"
)

var (
	ErrInvalidInterval = errors.New("agent: interval must be positive")
	ErrNoCollectors    = errors.New("agent: no collectors configured")
)

type Collector interface {
	Collect() ([]metric.Metric, error)
}

type Reporter interface {
	Report(metric.Metric) error
}

type Agent struct {
	interval   time.Duration
	reporter   Reporter
	collectors []Collector
	logger     *log.Logger
}

func New(interval time.Duration, reporter Reporter, collectors ...Collector) *Agent {
	return &Agent{
		interval:   interval,
		reporter:   reporter,
		collectors: collectors,
		logger:     log.Default(),
	}
}

func (a *Agent) Run(ctx context.Context) error {
	if a.interval <= 0 {
		return fmt.Errorf("%w: got %s", ErrInvalidInterval, a.interval)
	}

	if len(a.collectors) == 0 {
		return ErrNoCollectors
	}

	if err := a.collectOnce(); err != nil {
		return fmt.Errorf("agent: initial collect: %w", err)
	}

	ticker := time.NewTicker(a.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := a.collectOnce(); err != nil {
				a.logger.Printf("agent: %v", err)
			}
		}
	}
}

func (a *Agent) collectOnce() error {
	var errs []error
	for _, c := range a.collectors {
		metrics, err := c.Collect()
		if err != nil {
			errs = append(errs, err)
			continue
		}
		for _, m := range metrics {
			if err := a.reporter.Report(m); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}
