package procfs

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const minCPUFields = 4

var ErrNoCPULine = errors.New("procfs: aggregate cpu line not found in stat")

type CPUStat struct {
	User, Nice, System, Idle, IOWait, IRQ, SoftIRQ, Steal uint64
}

func ReadCPUStat(procRoot string) (CPUStat, error) {
	f, err := os.Open(filepath.Join(procRoot, "stat"))
	if err != nil {
		return CPUStat{}, err
	}
	defer f.Close()
	return ParseCPUStat(f)
}

func ParseCPUStat(r io.Reader) (CPUStat, error) {
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) > 0 && fields[0] == "cpu" {
			return parseCPULine(fields[1:])
		}
	}
	if err := sc.Err(); err != nil {
		return CPUStat{}, err
	}
	return CPUStat{}, ErrNoCPULine
}

func parseCPULine(values []string) (CPUStat, error) {
	if len(values) < minCPUFields {
		return CPUStat{}, fmt.Errorf("procfs: too few cpu fields: %d", len(values))
	}
	var s CPUStat
	dst := []*uint64{&s.User, &s.Nice, &s.System, &s.Idle, &s.IOWait, &s.IRQ, &s.SoftIRQ, &s.Steal}
	for i := 0; i < len(dst) && i < len(values); i++ {
		n, err := strconv.ParseUint(values[i], 10, 64)
		if err != nil {
			return CPUStat{}, fmt.Errorf("procfs: cpu field %d: %w", i, err)
		}
		*dst[i] = n
	}
	return s, nil
}

func (s CPUStat) UsageSince(prev CPUStat) (percent float64, ok bool) {
	idle := delta(s.Idle, prev.Idle) + delta(s.IOWait, prev.IOWait)
	busy := delta(s.User, prev.User) + delta(s.Nice, prev.Nice) +
		delta(s.System, prev.System) + delta(s.IRQ, prev.IRQ) +
		delta(s.SoftIRQ, prev.SoftIRQ) + delta(s.Steal, prev.Steal)

	total := idle + busy
	if total == 0 {
		return 0, false
	}
	return float64(busy) / float64(total) * 100, true
}

// delta returns cur-prev, or 0 if the counter went backwards.
func delta(cur, prev uint64) uint64 {
	if cur < prev {
		return 0
	}
	return cur - prev
}
