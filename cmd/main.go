package main

import (
    "context"
    "flag"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"

    "sentinel/monitor"
    "sentinel/proc"
)

func main() {
    interval := flag.Duration("interval", 1*time.Second, "refresh interval (e.g. 500ms, 1s)")
    hz := flag.Int("hz", 0, "clock ticks per second (0=auto-detect)")
    flag.Parse()

    logger := log.New(os.Stderr, "[sentinel] ", log.LstdFlags)

    // Auto-detect HZ if not specified
    actualHZ := *hz
    if actualHZ == 0 {
        actualHZ = proc.DetectHZ()
        logger.Printf("auto-detected HZ: %d", actualHZ)
    }

    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    engine := monitor.NewEngine()
    if err := engine.Run(ctx, *interval, actualHZ, logger); err != nil && err != context.Canceled {
        logger.Printf("engine exit with error: %v", err)
        os.Exit(1)
    }
}