package cli

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sothoth-nodeagent/pkg/config"
	"sothoth-nodeagent/pkg/naserver"
	"sothoth-nodeagent/pkg/util"
	"sothoth-nodeagent/pkg/xdaemon"
	"strconv"
	"syscall"
)

func init() {
	cfg := config.GetInstance()

	flag.StringVar(&cfg.ServerURL, "server", "", "Server URL")
	flag.StringVar(&cfg.ProjectID, "projectId", "", "Project ID")
	flag.StringVar(&cfg.NodeID, "nodeId", "", "Node ID")
	flag.StringVar(&cfg.SothothDir, "sothothDir", "/sothoth", "Sothoth directory")
	flag.BoolVar(&cfg.DaemonMode, "d", false, "Run as daemon (background)")
	flag.BoolVar(&cfg.Proxy, "p", false, "Enable proxy mode")
	flag.BoolVar(&cfg.Version, "version", false, "Show version")
	flag.StringVar(&cfg.Logflags, "logflags", "log.LstdFlags", "Log flags")
}

func ParseMain() {
	flag.Parse()
	cfg := config.GetInstance()
	if cfg.Version {
		fmt.Println("Sothoth NodeAgent v1.0.0 (Go)")
		return
	}

	if cfg.ProjectID == "" || cfg.NodeID == "" || cfg.ServerURL == "" {
		log.Fatal("Usage: sothoth-nodeagent -projectId <PROJECT_ID> -nodeId <NODE_ID> -server <SERVER_URL> [-sothothDir <DIR>] [-d] [-p]")
	}

	EnvInit(cfg)

	// 检查nodeagent.pid文件，判断是否已经有进程在运行中
	if checkRunning(cfg) {
		log.Fatal("Another instance of the agent is already running.")
		return
	}

	// Handle daemon mode
	if cfg.DaemonMode {
		logFile := filepath.Join(cfg.SothothDir, "logs/nodeagent/000000000000/agent.log")
		xdaemon.Background(logFile, true)
	}

	// 创建nodeagent.pid文件
	createPidFile(cfg)

	nodeAgent, err := naserver.NewNodeAgent(cfg.ProjectID, cfg.NodeID, cfg.ServerURL, cfg.SothothDir, cfg.Proxy)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down agent...")
		nodeAgent.Stop()
	}()

	// Start the agent
	if err := nodeAgent.Run(); err != nil {
		log.Fatalf("Agent failed: %v", err)
	}
}

func EnvInit(cfg *config.Config) {
	// 初始化各个目录
	// sothoth dir, mkdirwithperm 777
	if err := util.MkdirWithPerm(cfg.SothothDir, 0777); err != nil {
		log.Fatalf("Failed to create sothoth directory: %v", err)
	}

	// log file path
	logDir := filepath.Join(cfg.SothothDir, "logs/nodeagent/000000000000/")
	if err := util.MkdirWithPerm(logDir, 0777); err != nil {
		log.Fatalf("Failed to create log directory: %v", err)
	}

	// tmp file path
	tmpDir := filepath.Join(cfg.SothothDir, "tmp")
	if err := util.MkdirWithPerm(tmpDir, 0777); err != nil {
		log.Fatalf("Failed to create tmp directory: %v", err)
	}

	// 初始化环境变量
	// TMOUT/HISTSIZE/HISTFILE/TERM/PATH
	envVars := map[string]string{
		"TMOUT":    "0",                                            // 禁用shell超时
		"HISTSIZE": "1000",                                         // 历史命令数量
		"HISTFILE": filepath.Join(cfg.SothothDir, ".bash_history"), // 历史文件路径
		"PATH":     os.Getenv("PATH") + ":" + cfg.SothothDir,       // 添加sothoth bin目录到PATH
	}

	for key, value := range envVars {
		if err := os.Setenv(key, value); err != nil {
			log.Printf("Warning: Failed to set environment variable %s: %v", key, err)
		}
	}

	// TERM环境变量如果不存在再设置，如果已经设置了就不覆盖
	if os.Getenv("TERM") == "" {
		if err := os.Setenv("TERM", "xterm-256color"); err != nil {
			log.Printf("Warning: Failed to set TERM environment variable: %v", err)
		}
	}

	log.Printf("Environment initialized: sothoth_dir=%s", cfg.SothothDir)
}

func checkRunning(cfg *config.Config) bool {
	pidFile := filepath.Join(cfg.SothothDir, "nodeagent.pid")
	data, err := ioutil.ReadFile(pidFile)
	if err != nil {
		return false
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	err = process.Signal(syscall.Signal(0))
	return err == nil
}

func createPidFile(cfg *config.Config) {
	pidFile := filepath.Join(cfg.SothothDir, "nodeagent.pid")
	pid := os.Getpid()
	if err := ioutil.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0644); err != nil {
		log.Fatalf("Failed to create PID file: %v", err)
	}
}
