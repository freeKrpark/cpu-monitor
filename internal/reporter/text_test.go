package reporter_test

import (
	"bytes"
	"cpu-monitor/internal/agent"
	"cpu-monitor/internal/metric"
	"cpu-monitor/internal/reporter"
	"testing"
)

// Compile-time check that Text satisfies agent.Reporter.
var _ agent.Reporter = (*reporter.Text)(nil)

func TestTextReport(t *testing.T) {
	var buf bytes.Buffer
	r := reporter.NewText(&buf, "minikube")

	for _, m := range []metric.Metric{{Name: "CPU", Value: 12.5}, {Name: "CPU", Value: 8.44}} {
		if err := r.Report(m); err != nil {
			t.Fatal(err)
		}
	}

	want := "[Host: minikube] CPU: 12.5%\n[Host: minikube] CPU: 8.4%\n"
	if got := buf.String(); got != want {
		t.Errorf("output:\n%q\nwant:\n%q", got, want)
	}
}
