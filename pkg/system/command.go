package system

import (
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// CommandResult represents command execution result
type CommandResult struct {
	ExitCode      int    `json:"exit_code"`
	Stdout        string `json:"stdout"`
	Stderr        string `json:"stderr"`
	ExecutionTime int64  `json:"execution_time"`
}

// ExecuteCommand executes a system command and returns the result
func ExecuteCommand(command string) (*CommandResult, error) {
	startTime := time.Now()

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/C", command)
	default:
		cmd = exec.Command("sh", "-c", command)
	}

	// Set timeout
	timeout := 30 * time.Second
	done := make(chan error, 1)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		executionTime := time.Since(startTime).Milliseconds()

		exitCode := 0
		if err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				exitCode = exitError.ExitCode()
			} else {
				exitCode = -1
			}
		}

		return &CommandResult{
			ExitCode:      exitCode,
			Stdout:        stdout.String(),
			Stderr:        stderr.String(),
			ExecutionTime: executionTime,
		}, nil

	case <-time.After(timeout):
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return &CommandResult{
			ExitCode:      -1,
			Stdout:        "",
			Stderr:        "Command timeout",
			ExecutionTime: timeout.Milliseconds(),
		}, nil
	}
}
