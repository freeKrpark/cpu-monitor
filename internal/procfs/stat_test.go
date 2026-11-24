package procfs

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestParseCPUStat(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    CPUStat
		wantErr error
	}{
		{
			name:  "ignores guest fields and per-core lines",
			input: "cpu  100 1 50 800 50 2 3 4 30 5\ncpu0 9 9 9 9 9 9 9 9 0 0\n",
			want:  CPUStat{User: 100, Nice: 1, System: 50, Idle: 800, IOWait: 50, IRQ: 2, SoftIRQ: 3, Steal: 4},
		},
		{
			name:  "old kernel with 4 fields",
			input: "cpu 10 0 10 80\n",
			want:  CPUStat{User: 10, System: 10, Idle: 80},
		},
		{name: "missing cpu line", input: "intr 1\nctxt 2\n", wantErr: ErrNoCPULine},
		{name: "empty input", input: "", wantErr: ErrNoCPULine},
		{name: "too few fields", input: "cpu 1 2\n", wantErr: errAny},
		{name: "non-numeric field", input: "cpu 1 x 3 4\n", wantErr: strconv.ErrSyntax},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCPUStat(strings.NewReader(tt.input))
			assertErr(t, err, tt.wantErr)
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestReadCPUStat(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "stat"), []byte("cpu 10 0 10 80 0 0 0 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := ReadCPUStat(root)
	if err != nil {
		t.Fatal(err)
	}
	if want := (CPUStat{User: 10, System: 10, Idle: 80}); got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}

	_, err = ReadCPUStat(filepath.Join(root, "missing"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("err = %v, want os.ErrNotExist", err)
	}
}

func TestUsageSince(t *testing.T) {
	prev := CPUStat{User: 100, System: 100, Idle: 750, IOWait: 50}
	tests := []struct {
		name   string
		cur    CPUStat
		want   float64
		wantOK bool
	}{
		{"25 percent busy", CPUStat{User: 120, System: 105, Idle: 825, IOWait: 50}, 25, true},
		{"iowait counts as idle", CPUStat{User: 100, System: 100, Idle: 750, IOWait: 150}, 0, true},
		{"fully busy", CPUStat{User: 200, System: 100, Idle: 750, IOWait: 50}, 100, true},
		// Regression: busy counter going backwards used to yield a negative value.
		{"busy counter backwards never negative", CPUStat{User: 90, System: 100, Idle: 850, IOWait: 50}, 0, true},
		// iowait going backwards must not cancel out real idle time.
		{"iowait backwards is clamped", CPUStat{User: 125, System: 100, Idle: 825, IOWait: 40}, 25, true},
		{"idle backwards never above 100", CPUStat{User: 110, System: 100, Idle: 740, IOWait: 50}, 100, true},
		{"no elapsed time", prev, 0, false},
		{"full counter reset", CPUStat{User: 1, Idle: 1}, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := tt.cur.UsageSince(prev)
			if ok != tt.wantOK || math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("got (%v, %v), want (%v, %v)", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// FuzzUsageSince checks the invariant that utilization is always within
// [0, 100] for arbitrary counter values, including ones that go backwards.
func FuzzUsageSince(f *testing.F) {
	f.Add(uint64(100), uint64(800), uint64(50), uint64(90), uint64(850), uint64(40))
	f.Fuzz(func(t *testing.T, pu, pi, pw, cu, ci, cw uint64) {
		prev := CPUStat{User: pu, Idle: pi, IOWait: pw}
		cur := CPUStat{User: cu, Idle: ci, IOWait: cw}
		if got, ok := cur.UsageSince(prev); ok && (got < 0 || got > 100 || math.IsNaN(got)) {
			t.Fatalf("UsageSince(%+v, %+v) = %v, out of [0, 100]", prev, cur, got)
		}
	})
}

var errAny = errors.New("any error")

func assertErr(t *testing.T, err, want error) {
	t.Helper()
	switch {
	case want == nil && err != nil:
		t.Fatalf("unexpected error: %v", err)
	case want == errAny && err == nil:
		t.Fatal("expected an error")
	case want != nil && want != errAny && !errors.Is(err, want):
		t.Fatalf("err = %v, want %v", err, want)
	}
}
