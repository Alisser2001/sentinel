package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
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
		// subcommands: start | run | stop | status
		if len(os.Args) < 3 {
			fmt.Println("usage: sentinel daemon <start|stop|status>")
			os.Exit(2)
		}
		sub := os.Args[2]
		switch sub {
		case "start":
			startDaemon(hz)
		case "run":
			runDaemon(hz)
		case "stop":
			stopDaemon()
		case "status":
			statusDaemon()
		default:
			fmt.Println("unknown daemon subcommand:", sub)
			os.Exit(2)
		}

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

// runDaemon runs the daemon in the foreground. Used by the background child process.
func runDaemon(hz int) {
	// Graceful shutdown on SIGINT/SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger := log.New(os.Stderr, "[sentinel-daemon] ", log.LstdFlags)
	d := daemon.New(1*time.Second, hz, logger)
	_ = d.Run(ctx)
}

// startDaemon starts a detached background process and exits 0 immediately.
func startDaemon(hz int) {
	pidPath := daemonPIDPath()

	// If already running, don't start another
	if pid, err := readPID(pidPath); err == nil && pid > 0 {
		if processExists(pid) {
			fmt.Println("daemon already running (pid:", pid, ")")
			os.Exit(0)
		}
	}

	// Launch a detached child process: sentinel daemon run
	exe, _ := os.Executable()
	cmd := exec.Command(exe, "daemon", "run")
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	// Detach from parent session
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "failed to start daemon:", err)
		os.Exit(1)
	}

	// Write child PID and exit 0
	_ = writePID(pidPath, cmd.Process.Pid)
	fmt.Println("daemon started (pid:", cmd.Process.Pid, ")")
	os.Exit(0)
}

func stopDaemon() {
	pidPath := daemonPIDPath()
	pid, err := readPID(pidPath)
	if err != nil || pid <= 0 {
		fmt.Println("daemon not running")
		os.Exit(0)
	}

	// Send SIGTERM
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		// If process not found, treat as stopped
		if !errors.Is(err, syscall.ESRCH) {
			fmt.Fprintln(os.Stderr, "failed to stop daemon:", err)
			os.Exit(1)
		}
	}

	// Best-effort cleanup
	_ = os.Remove(pidPath)
	fmt.Println("daemon stopped")
}

func statusDaemon() {
	pidPath := daemonPIDPath()
	pid, err := readPID(pidPath)
	if err != nil || pid <= 0 || !processExists(pid) {
		fmt.Println("daemon: stopped")
		return
	}
	fmt.Println("daemon: running (pid:", pid, ")")
}

// PID file helpers
func daemonPIDPath() string {
	dir := filepath.Join(os.TempDir(), "sentinel")
	_ = os.MkdirAll(dir, 0o755)
	return filepath.Join(dir, "daemon.pid")
}

func writePID(path string, pid int) error {
	return os.WriteFile(path, []byte(strconv.Itoa(pid)), 0o644)
}

func readPID(path string) (int, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(string(b))
	if err != nil {
		return 0, err
	}
	return pid, nil
}

func processExists(pid int) bool {
	// Signal 0 checks for existence without sending a real signal
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}
