// Package cli 提供命令行参数解析和程序初始化功能
//
// 该包负责：
// - 解析命令行参数
// - 初始化运行环境
// - 管理进程生命周期
// - 处理优雅关闭
package cli

import (
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
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

// init 函数在包加载时自动执行，用于初始化命令行参数
func init() {
	cfg := config.GetInstance()

	// 定义命令行参数
	flag.StringVar(&cfg.Server, "server", "", "Sothoth Server WebSocket URL")
	flag.StringVar(&cfg.ProjectID, "projectId", "", "Project ID")
	flag.StringVar(&cfg.NodeID, "nodeId", "", "Node ID")
	flag.StringVar(&cfg.SothothDir, "sothothDir", "/sothoth", "Sothoth工作目录")
	flag.BoolVar(&cfg.DaemonMode, "d", false, "daemon(background)")
	flag.BoolVar(&cfg.ProxyMode, "p", false, "enable proxy mode")
	flag.BoolVar(&cfg.Version, "version", false, "version")
	flag.StringVar(&cfg.Logflags, "logflags", "log.LstdFlags", "logflag")
	flag.StringVar(&cfg.Socks5Addr, "socks5", "", "socks5 proxy addr, eg. 127.0.0.1:1080")
	flag.BoolVar(&cfg.AutoHook, "autohook", false, "monitor and auto hook all match process")
}

// ParseMain 解析命令行参数并启动NodeAgent
// 这是程序的主要入口函数，负责：
// - 解析和验证命令行参数
// - 初始化运行环境
// - 检查进程唯一性
// - 启动NodeAgent并处理优雅关闭
func ParseMain() {
	flag.Parse()
	cfg := config.GetInstance()

	// 如果请求显示版本信息，则输出版本并退出
	if cfg.Version {
		fmt.Println("Sothoth NodeAgent v1.0.0 (Go)")
		return
	}

	// 验证必需的命令行参数
	if cfg.ProjectID == "" || cfg.NodeID == "" || cfg.Server == "" {
		log.Fatal("Usage: sothoth-nodeagent -projectId <PROJECT_ID> -nodeId <NODE_ID> -server <SERVER_URL> [-sothothDir <DIR>] [-d] [-p]")
	}

	// 初始化运行环境
	EnvInit(cfg)

	// 检查nodeagent.pid文件，判断是否已经有进程在运行中
	if checkRunning(cfg) {
		log.Fatal("Another agent is running.")
		return
	}

	// 处理守护进程模式
	if cfg.DaemonMode {
		logFile := filepath.Join(cfg.SothothDir, "logs/nodeagent/000000000000/agent.log")
		xdaemon.Background(logFile, true)
	}

	// 创建nodeagent.pid文件
	createPidFile(cfg)

	// 创建NodeAgent实例
	nodeAgent, err := naserver.NewNodeAgent()
	if err != nil {
		log.Fatalf("create Agent failed: %v", err)
	}

	// 启动Agent
	if err := nodeAgent.Start(); err != nil {
		log.Fatalf("Agent start failed: %v", err)
	}

	// 处理优雅关闭
	firstInterrupt := true
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	for { // accept multiple signal
		select {
		case <-signalChan:
			if firstInterrupt {
				log.Println("press CTRL+C to force exit")
				firstInterrupt = false
				log.Info("Stoping agent...")
				nodeAgent.Stop()
			} else {
				os.Exit(0)
			}
		}
	}
}

// EnvInit 初始化NodeAgent运行环境
// 创建必要的目录结构并设置环境变量，包括：
// - Sothoth工作目录
// - 日志目录
// - 临时文件目录
// - Shell环境变量配置
func EnvInit(cfg *config.Config) {
	// 初始化各个目录
	// 创建sothoth主目录，权限设置为777
	if err := util.MkdirWithPerm(cfg.SothothDir, 0777); err != nil {
		log.Fatalf("create sothoth dir error: %v", err)
	}

	// 创建日志文件目录
	logDir := filepath.Join(cfg.SothothDir, "logs/nodeagent/000000000000/")
	if err := util.MkdirWithPerm(logDir, 0777); err != nil {
		log.Fatalf("create log dir error: %v", err)
	}

	// 创建临时文件目录
	tmpDir := filepath.Join(cfg.SothothDir, "tmp")
	if err := util.MkdirWithPerm(tmpDir, 0777); err != nil {
		log.Fatalf("create tmp dir error: %v", err)
	}

	// 初始化环境变量
	// 设置Shell相关的环境变量以优化终端体验
	envVars := map[string]string{
		"TMOUT":    "0",                                            // 禁用shell超时
		"HISTSIZE": "1000",                                         // 历史命令数量
		"HISTFILE": filepath.Join(cfg.SothothDir, ".bash_history"), // 历史文件路径
		"PATH":     os.Getenv("PATH") + ":" + cfg.SothothDir,       // 添加sothoth bin目录到PATH
	}

	for key, value := range envVars {
		if err := os.Setenv(key, value); err != nil {
			log.Warnf("set env %s error: %v", key, err)
		}
	}

	// TERM环境变量如果不存在再设置，如果已经设置了就不覆盖
	if os.Getenv("TERM") == "" {
		if err := os.Setenv("TERM", "xterm-256color"); err != nil {
			log.Warnf("set env TERM error: %v", err)
		}
	}

	log.Infof("EnvInit success: sothoth_dir=%s", cfg.SothothDir)
}

// checkRunning 检查是否已有NodeAgent实例在运行
// 通过读取PID文件并检查进程是否存在来判断
// 返回true表示已有实例在运行，false表示没有
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

	// 发送信号0来检查进程是否存在
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// createPidFile 创建PID文件
// 将当前进程的PID写入到指定的PID文件中，用于进程管理和唯一性检查
func createPidFile(cfg *config.Config) {
	pidFile := filepath.Join(cfg.SothothDir, "nodeagent.pid")
	pid := os.Getpid()
	if err := ioutil.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0644); err != nil {
		log.Fatalf("create pid file error: %v", err)
	}
}
