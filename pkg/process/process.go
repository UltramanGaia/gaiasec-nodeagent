package process

import (
	"context"
	"gaiasec-nodeagent/pkg/network"
	"gaiasec-nodeagent/pkg/pb"
	"strings"

	"github.com/shirou/gopsutil/v3/process"
)

// ProcessInfo represents process information
type ProcessInfo struct {
	PID     int    `json:"pid"`
	PPID    int    `json:"ppid"`
	Name    string `json:"name"`
	Cmdline string `json:"cmdline"`
	User    string `json:"user"`
}

// GetProcessList returns a list of running processes using gopsutil
func GetProcessList() ([]*pb.Process, error) {
	ctx := context.Background()

	// Get all process PIDs
	pids, err := process.Pids()
	if err != nil {
		return nil, err
	}

	// Get network connections to check for listening ports
	connections, err := network.GetNetworkConnections()
	if err != nil {
		// If we can't get network connections, just continue without this information
		// We don't want to fail the entire process collection
	}

	// Create a map of PID to listening status
	listeningPorts := make(map[int32]bool)
	for _, conn := range connections {
		// Check if this is a listening connection
		// For TCP, listening state is indicated by remote address being 0.0.0.0:0 or [::]:0
		// For UDP, all connections are effectively listening
		if conn.RemoteAddress == "0.0.0.0" && conn.RemotePort == 0 || conn.RemoteAddress == "::" && conn.RemotePort == 0 {
			listeningPorts[conn.Pid] = true
		}
	}

	var processes []*pb.Process

	for _, pid := range pids {
		proc, err := process.NewProcess(pid)
		if err != nil {
			continue // Skip processes we can't access
		}

		processInfo, err := getProcessInfo(ctx, proc)
		if err != nil {
			continue // Skip processes we can't read
		}

		// Set HasListeningPort field
		if _, ok := listeningPorts[processInfo.Pid]; ok {
			processInfo.IsListen = true
		}

		processes = append(processes, processInfo)
	}

	return processes, nil
}

// getProcessInfo extracts process information using gopsutil
func getProcessInfo(ctx context.Context, proc *process.Process) (*pb.Process, error) {
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

	// Get process user
	user, err := proc.Username()
	if err != nil {
		user = "unknown"
	}

	return &pb.Process{
		Pid:     pid,
		Ppid:    ppid,
		Name:    name,
		Cmdline: cmdline,
		User:    user,
	}, nil
}
