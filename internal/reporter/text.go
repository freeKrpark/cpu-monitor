package reporter

import (
	"cpu-monitor/internal/metric"
	"fmt"
	"io"
)

type Text struct {
	w    io.Writer
	host string
}

func NewText(w io.Writer, host string) *Text {
	return &Text{w: w, host: host}
}

func (t *Text) Report(m metric.Metric) error {
	_, err := fmt.Fprintf(t.w, "[Host: %s] %s: %.1f%%\n", t.host, m.Name, m.Value)
	return err
}
