package collector

import (
	"cpu-monitor/internal/metric"
	"cpu-monitor/internal/procfs"
	"fmt"
)

type CPU struct {
	procRoot string
	prev     procfs.CPUStat
	hasPrev  bool
}

func NewCPU(procRoot string) *CPU {
	return &CPU{procRoot: procRoot}
}

func (c *CPU) Collect() ([]metric.Metric, error) {
	cur, err := procfs.ReadCPUStat(c.procRoot)
	if err != nil {
		return nil, fmt.Errorf("cpu: %w", err)
	}

	prev, hadPrev := c.prev, c.hasPrev
	c.prev, c.hasPrev = cur, true

	if !hadPrev {
		return nil, nil // baseline only
	}
	usage, ok := cur.UsageSince(prev)
	if !ok {
		return nil, nil // no elapsed time or counter reset
	}
	return []metric.Metric{{Name: "CPU", Value: usage}}, nil
}
