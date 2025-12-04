package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"sentinel/daemon"
	"sentinel/monitor"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}

	cmd := os.Args[1]

	switch cmd {

	case "tui":
		runTUI()

	case "daemon":
		runDaemon()

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

func runTUI() {
	ctx := context.Background()
	interval := 1500 * time.Millisecond

	engine := monitor.NewEngine()
	engine.Run(ctx, interval, 100, log.New(os.Stderr, "[sentinel] ", log.LstdFlags))
}

func runDaemon() {
	ctx := context.Background()
	logger := log.New(os.Stderr, "[sentinel-daemon] ", log.LstdFlags)

	d := daemon.New(1*time.Second, 100, logger)
	d.Run(ctx)
}