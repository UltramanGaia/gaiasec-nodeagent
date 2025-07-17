package system

import (
	"context"
	"strings"

	"github.com/shirou/gopsutil/v3/process"
)

// ProcessInfo represents process information
type ProcessInfo struct {
	PID         int    `json:"pid"`
	PPID        int    `json:"ppid"`
	Comm        string `json:"comm"`
	CommandLine string `json:"command_line"`
}

// GetProcessList returns a list of running processes using gopsutil
func GetProcessList() ([]ProcessInfo, error) {
	ctx := context.Background()
	
	// Get all process PIDs
	pids, err := process.Pids()
	if err != nil {
		return nil, err
	}

	var processes []ProcessInfo

	for _, pid := range pids {
		proc, err := process.NewProcess(pid)
		if err != nil {
			continue // Skip processes we can't access
		}

		processInfo, err := getProcessInfo(ctx, proc)
		if err != nil {
			continue // Skip processes we can't read
		}

		processes = append(processes, *processInfo)
	}

	return processes, nil
}

// getProcessInfo extracts process information using gopsutil
func getProcessInfo(ctx context.Context, proc *process.Process) (*ProcessInfo, error) {
	pid := proc.Pid

	// Get parent PID
	ppid, err := proc.Ppid()
	if err != nil {
		ppid = 0 // Set to 0 if we can't get PPID
	}

	// Get process name
	name, err := proc.Name()
	if err != nil {
		name = "unknown"
	}

	// Get command line
	cmdline, err := proc.Cmdline()
	if err != nil {
		// If we can't get cmdline, try to get exe path as fallback
		exe, exeErr := proc.Exe()
		if exeErr != nil {
			cmdline = name // Final fallback to process name
		} else {
			cmdline = exe
		}
	}

	// Clean up command line
	cmdline = strings.TrimSpace(cmdline)
	if cmdline == "" {
		cmdline = "[" + name + "]"
	}

	return &ProcessInfo{
		PID:         int(pid),
		PPID:        int(ppid),
		Comm:        name,
		CommandLine: cmdline,
	}, nil
}
