package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"sentinel/daemon"
	"sentinel/monitor"
	"sentinel/proc"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}
	hz := proc.DetectHZ()
	cmd := os.Args[1]

	switch cmd {

	case "tui":
		runTUI(hz)

	case "daemon":
		runDaemon(hz)

	case "help":
		usage()

	default:
		fmt.Println("unknown command:", cmd)
		usage()
	}
}

func usage() {
	fmt.Println(`
        Sentinel commands:
        sentinel tui       → start the TUI monitor
        sentinel daemon    → start background alert daemon
        sentinel help      → show help
    `)
}

func runTUI(hz int) {
	ctx := context.Background()
	interval := 1500 * time.Millisecond

	engine := monitor.NewEngine()
	engine.Run(ctx, interval, hz, log.New(os.Stderr, "[sentinel] ", log.LstdFlags))
}

func runDaemon(hz int) {
	ctx := context.Background()
	logger := log.New(os.Stderr, "[sentinel-daemon] ", log.LstdFlags)

	d := daemon.New(1*time.Second, hz, logger)
	d.Run(ctx)
}
