package process

import (
	"context"
	"fmt"
	"gaiasec-nodeagent/pkg/pb"
	"path/filepath"
	"sort"
	"strings"

	"github.com/shirou/gopsutil/v3/process"
)

const (
	maxEnvVars        = 256
	maxEnvKeyLength   = 128
	maxEnvValueLength = 2048
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

		processes = append(processes, processInfo)
	}

	return processes, nil
}

// GetProcessMetadata returns detailed metadata for a single process.
func GetProcessMetadata(pid int32) (*pb.ProcessMetadataResponse, error) {
	ctx := context.Background()
	proc, err := process.NewProcess(pid)
	if err != nil {
		return nil, err
	}

	processInfo, err := getProcessInfo(ctx, proc)
	if err != nil {
		return nil, err
	}

	executable, err := proc.Exe()
	if err != nil {
		executable = ""
	}
	executable = sanitizePath(executable)

	cwd, err := proc.Cwd()
	if err != nil {
		cwd = ""
	}
	cwd = sanitizePath(cwd)

	envVars := collectEnvVars(proc)
	cmdlineArgs, err := proc.CmdlineSlice()
	if err != nil || len(cmdlineArgs) == 0 {
		cmdlineArgs = splitCommandLine(processInfo.GetCmdline())
	}
	jvmArgs, mainClass, jarPath := parseJavaCommand(cmdlineArgs)

	listenPorts, err := collectListenPorts(proc)
	if err != nil {
		listenPorts = nil
	}

	startSignature, err := buildStartSignature(proc)
	if err != nil {
		startSignature = ""
	}

	return &pb.ProcessMetadataResponse{
		Pid:            pid,
		Name:           processInfo.GetName(),
		Cmdline:        processInfo.GetCmdline(),
		Executable:     executable,
		Cwd:            cwd,
		User:           processInfo.GetUser(),
		JvmArgs:        jvmArgs,
		MainClass:      mainClass,
		JarPath:        jarPath,
		EnvVars:        envVars,
		ListenPorts:    listenPorts,
		StartSignature: startSignature,
	}, nil
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
	name = sanitizeProcessString(name, "unknown")

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
	cmdline = sanitizeProcessString(cmdline, "["+name+"]")

	// Get process user
	user, err := proc.Username()
	if err != nil {
		user = "unknown"
	}
	user = sanitizeProcessString(user, "unknown")

	return &pb.Process{
		Pid:     pid,
		Ppid:    ppid,
		Name:    name,
		Cmdline: cmdline,
		User:    user,
	}, nil
}

func sanitizeProcessString(value string, fallback string) string {
	value = strings.TrimSpace(strings.ToValidUTF8(value, ""))
	if value == "" {
		return fallback
	}

	return value
}

func sanitizePath(value string) string {
	value = sanitizeProcessString(value, "")
	if value == "" {
		return ""
	}
	return filepath.Clean(value)
}

func collectEnvVars(proc *process.Process) map[string]string {
	values, err := proc.Environ()
	if err != nil || len(values) == 0 {
		return nil
	}

	envVars := make(map[string]string)
	for _, item := range values {
		if len(envVars) >= maxEnvVars {
			break
		}
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		key = sanitizeProcessString(key, "")
		if key == "" {
			continue
		}
		key = truncateString(key, maxEnvKeyLength)
		value = truncateString(sanitizeProcessString(value, ""), maxEnvValueLength)
		envVars[key] = value
	}
	if len(envVars) == 0 {
		return nil
	}
	return envVars
}

func collectListenPorts(proc *process.Process) ([]int32, error) {
	connections, err := proc.Connections()
	if err != nil {
		return nil, err
	}

	ports := make(map[uint32]struct{})
	for _, connection := range connections {
		if connection.Status != "LISTEN" || connection.Laddr.Port == 0 {
			continue
		}
		ports[connection.Laddr.Port] = struct{}{}
	}
	if len(ports) == 0 {
		return nil, nil
	}

	ordered := make([]int, 0, len(ports))
	for port := range ports {
		ordered = append(ordered, int(port))
	}
	sort.Ints(ordered)

	result := make([]int32, 0, len(ordered))
	for _, port := range ordered {
		result = append(result, int32(port))
	}
	return result, nil
}

func buildStartSignature(proc *process.Process) (string, error) {
	createdAt, err := proc.CreateTime()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d", createdAt), nil
}

func parseJavaCommand(args []string) ([]string, string, string) {
	if len(args) <= 1 {
		return nil, "", ""
	}

	jvmArgs := make([]string, 0)
	mainClass := ""
	jarPath := ""

	for i := 1; i < len(args); i++ {
		arg := sanitizeProcessString(args[i], "")
		if arg == "" {
			continue
		}
		if jarPath == "" && arg == "-jar" && i+1 < len(args) {
			jarPath = sanitizePath(args[i+1])
			break
		}
		if strings.HasPrefix(arg, "-") {
			jvmArgs = append(jvmArgs, arg)
			if takesValue(arg) && i+1 < len(args) {
				value := sanitizeProcessString(args[i+1], "")
				if value != "" {
					jvmArgs = append(jvmArgs, value)
				}
				i++
			}
			continue
		}
		mainClass = arg
		break
	}

	return jvmArgs, mainClass, jarPath
}

func takesValue(arg string) bool {
	switch arg {
	case "-cp", "-classpath", "-p", "--module-path":
		return true
	default:
		return false
	}
}

func splitCommandLine(command string) []string {
	command = strings.TrimSpace(command)
	if command == "" {
		return nil
	}
	return strings.Fields(command)
}

func truncateString(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[:limit]
}
