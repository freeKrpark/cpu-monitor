package main

import (
	"context"
	"cpu-monitor/internal/agent"
	"cpu-monitor/internal/collector"
	"cpu-monitor/internal/reporter"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const defaultInterval = 5 * time.Second

func main() {
	log.SetFlags(0)
	log.SetPrefix("cpu-monitor: ")
	procRoot := envOr("PROC_ROOT", "/proc")
	nodeName := envOr("NODE_NAME", "unknown")
	interval, err := time.ParseDuration(envOr("COLLECT_INTERVAL", defaultInterval.String()))
	if err != nil {
		log.Fatalf("invalid COLLECT_INTERVAL: %v", err)
	}

	a := agent.New(
		interval,
		reporter.NewText(os.Stdout, nodeName),
		collector.NewCPU(procRoot),
	)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := a.Run(ctx); err != nil {
		log.Fatal(err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
