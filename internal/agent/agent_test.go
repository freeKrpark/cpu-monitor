package agent

import (
	"context"
	"cpu-monitor/internal/metric"
	"errors"
	"io"
	"log"
	"sync"
	"testing"
	"time"
)

// fakeCollector returns the next scripted result on each call,
// repeating the last one once the script is exhausted.
type fakeCollector struct {
	mu      sync.Mutex
	results []result
	calls   int
}

type result struct {
	metrics []metric.Metric
	err     error
}

func (f *fakeCollector) Collect() ([]metric.Metric, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r := f.results[min(f.calls, len(f.results)-1)]
	f.calls++
	return r.metrics, r.err
}

// fakeReporter records reported metrics and signals each one on ch.
type fakeReporter struct {
	mu  sync.Mutex
	got []metric.Metric
	ch  chan struct{}
}

func newFakeReporter() *fakeReporter {
	return &fakeReporter{ch: make(chan struct{}, 100)}
}

func (f *fakeReporter) Report(m metric.Metric) error {
	f.mu.Lock()
	f.got = append(f.got, m)
	f.mu.Unlock()
	select { // never block the agent loop, even if the test stopped reading
	case f.ch <- struct{}{}:
	default:
	}
	return nil
}

func (f *fakeReporter) waitFor(t *testing.T, n int) []metric.Metric {
	t.Helper()
	for i := 0; i < n; i++ {
		select {
		case <-f.ch:
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for report %d/%d", i+1, n)
		}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]metric.Metric(nil), f.got...)
}

func newTestAgent(r Reporter, cs ...Collector) *Agent {
	a := New(time.Millisecond, r, cs...)
	a.logger = log.New(io.Discard, "", 0)
	return a
}

// start runs the agent in the background and returns a stop function
// that cancels it and returns Run's error.
func start(t *testing.T, a *Agent) (stop func() error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- a.Run(ctx) }()
	return func() error {
		cancel()
		select {
		case err := <-done:
			return err
		case <-time.After(time.Second):
			t.Fatal("Run did not return after cancel")
			return nil
		}
	}
}

func TestRunReportsEveryTick(t *testing.T) {
	cpu := metric.Metric{Name: "CPU", Value: 10}
	c := &fakeCollector{results: []result{
		{}, // first call: nothing yet (like a delta collector's baseline)
		{metrics: []metric.Metric{cpu}},
	}}
	r := newFakeReporter()

	stop := start(t, newTestAgent(r, c))
	got := r.waitFor(t, 3)
	if err := stop(); err != nil {
		t.Fatalf("Run returned %v, want nil on cancel", err)
	}

	for _, m := range got {
		if m != cpu {
			t.Errorf("reported %+v, want %+v", m, cpu)
		}
	}
}

func TestRunFailingCollectorDoesNotBlockOthers(t *testing.T) {
	ok := metric.Metric{Name: "OK", Value: 1}
	failing := &fakeCollector{results: []result{{}, {err: errors.New("boom")}}}
	healthy := &fakeCollector{results: []result{{metrics: []metric.Metric{ok}}}}
	r := newFakeReporter()

	stop := start(t, newTestAgent(r, failing, healthy))
	got := r.waitFor(t, 3) // first round + at least two ticks where failing errors
	if err := stop(); err != nil {
		t.Fatalf("Run returned %v, want nil", err)
	}

	for _, m := range got {
		if m != ok {
			t.Errorf("reported %+v, want %+v", m, ok)
		}
	}
}

func TestRunFailsFastOnInitialError(t *testing.T) {
	wantErr := errors.New("no /proc")
	c := &fakeCollector{results: []result{{err: wantErr}}}

	err := newTestAgent(newFakeReporter(), c).Run(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("Run = %v, want wrapping %v", err, wantErr)
	}
}

func TestRunRejectsInvalidConfig(t *testing.T) {
	c := &fakeCollector{results: []result{{}}}
	tests := []struct {
		name    string
		a       *Agent
		wantErr error
		wantMsg string
	}{
		{"zero interval", New(0, newFakeReporter(), c), ErrInvalidInterval, "agent: interval must be positive: got 0s"},
		{"negative interval", New(-time.Second, newFakeReporter(), c), ErrInvalidInterval, "agent: interval must be positive: got -1s"},
		{"no collectors", New(time.Second, newFakeReporter()), ErrNoCollectors, "agent: no collectors configured"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.a.Run(context.Background())
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if err.Error() != tt.wantMsg {
				t.Errorf("message = %q, want %q", err.Error(), tt.wantMsg)
			}
		})
	}
}
