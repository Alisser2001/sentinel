package proc

import (
	"fmt"
	"syscall"
)

// KillProcess sends a signal to the given PID
func KillProcess(pid int, sig syscall.Signal) error {
	if pid <= 0 {
		return fmt.Errorf("invalid PID: %d", pid)
	}

	if err := syscall.Kill(pid, sig); err != nil {
		return fmt.Errorf("failed to send signal %v to PID %d: %w", sig, pid, err)
	}

	return nil
}

// TerminateProcess sends SIGTERM (graceful shutdown)
func TerminateProcess(pid int) error {
	return KillProcess(pid, syscall.SIGTERM)
}

// ForceKillProcess sends SIGKILL (immediate termination)
func ForceKillProcess(pid int) error {
	return KillProcess(pid, syscall.SIGKILL)
}

// SetProcessPriority changes the nice value of a process
// nice: -20 (highest priority) to 19 (lowest priority)
// Requires root for negative values or other users' processes
func SetProcessPriority(pid int, nice int) error {
	if pid <= 0 {
		return fmt.Errorf("invalid PID: %d", pid)
	}

	if nice < -20 || nice > 19 {
		return fmt.Errorf("nice value must be between -20 and 19, got %d", nice)
	}

	// PRIO_PROCESS = 0 (from sys/resource.h)
	if err := syscall.Setpriority(syscall.PRIO_PROCESS, pid, nice); err != nil {
		return fmt.Errorf("failed to set priority for PID %d: %w", pid, err)
	}

	return nil
}

// GetProcessPriority returns current nice value
func GetProcessPriority(pid int) (int, error) {
	if pid <= 0 {
		return 0, fmt.Errorf("invalid PID: %d", pid)
	}

	prio, err := syscall.Getpriority(syscall.PRIO_PROCESS, pid)
	if err != nil {
		return 0, fmt.Errorf("failed to get priority for PID %d: %w", pid, err)
	}

	return prio, nil
}
