package collector_test

import (
	"cpu-monitor/internal/agent"
	"cpu-monitor/internal/collector"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// Compile-time check that CPU satisfies agent.Collector.
var _ agent.Collector = (*collector.CPU)(nil)

func writeStat(t *testing.T, root, line string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "stat"), []byte(line+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCPUCollect(t *testing.T) {
	root := t.TempDir()
	c := collector.NewCPU(root)

	// 1st call: baseline only.
	writeStat(t, root, "cpu 100 0 100 800 0 0 0 0")
	got, err := c.Collect()
	if err != nil || len(got) != 0 {
		t.Fatalf("first Collect = (%v, %v), want (empty, nil)", got, err)
	}

	// 2nd call: +100 jiffies, 75 of them idle -> 25% busy.
	writeStat(t, root, "cpu 120 0 105 875 0 0 0 0")
	got, err = c.Collect()
	if err != nil || len(got) != 1 {
		t.Fatalf("second Collect = (%v, %v), want one metric", got, err)
	}
	if got[0].Name != "CPU" || math.Abs(got[0].Value-25) > 1e-9 {
		t.Errorf("metric = %+v, want {CPU 25}", got[0])
	}
}

func TestCPUCollectCounterReset(t *testing.T) {
	root := t.TempDir()
	c := collector.NewCPU(root)

	writeStat(t, root, "cpu 1000 0 0 1000 0 0 0 0")
	if _, err := c.Collect(); err != nil {
		t.Fatalf("baseline Collect: %v", err)
	}

	writeStat(t, root, "cpu 10 0 0 10 0 0 0 0")
	if got, err := c.Collect(); err != nil || len(got) != 0 {
		t.Fatalf("after reset = (%v, %v), want (empty, nil)", got, err)
	}

	writeStat(t, root, "cpu 60 0 0 60 0 0 0 0") // +100, 50 idle -> 50%
	got, err := c.Collect()
	if err != nil || len(got) != 1 || math.Abs(got[0].Value-50) > 1e-9 {
		t.Fatalf("after recovery = (%v, %v), want 50%%", got, err)
	}
}

func TestCPUCollectMissingFile(t *testing.T) {
	c := collector.NewCPU(filepath.Join(t.TempDir(), "nope"))
	if _, err := c.Collect(); err == nil {
		t.Error("expected error for missing /proc/stat")
	}
}
