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
	"gaiasec-nodeagent/pkg/config"
	"gaiasec-nodeagent/pkg/naserver"
	"gaiasec-nodeagent/pkg/pb"
	"gaiasec-nodeagent/pkg/util"
	"gaiasec-nodeagent/pkg/version"
	"gaiasec-nodeagent/pkg/xdaemon"
	"io/ioutil"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	log "github.com/sirupsen/logrus"
)

// init 函数在包加载时自动执行，用于初始化命令行参数
func init() {
	cfg := config.GetInstance()

	// 定义命令行参数
	flag.StringVar(&cfg.Server, "server", "", "GaiaSec Server WebSocket URL")
	flag.StringVar(&cfg.ProjectID, "projectId", "", "Project ID")
	flag.StringVar(&cfg.NodeID, "nodeId", "", "Node ID")
	flag.StringVar(&cfg.GaiaSecDir, "gaiasecDir", "/gaiasec", "GaiaSec工作目录")
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
		fmt.Printf("GaiaSec NodeAgent %s (Go)\n", version.Version)
		return
	}

	// 启动时打印版本信息
	log.Infof("GaiaSec NodeAgent starting, version: %s", version.Version)

	// 验证必需的命令行参数
	if cfg.ProjectID == "" || cfg.NodeID == "" || cfg.Server == "" {
		log.Fatal("Usage: gaiasec-nodeagent -projectId <PROJECT_ID> -nodeId <NODE_ID> -server <SERVER_URL> [-gaiasecDir <DIR>] [-d] [-p]")
	}

	// 设置自定义日志格式化器，与JavaAgent格式保持一致
	log.SetFormatter(&JavaAgentStyleFormatter{
		NodeID: cfg.NodeID,
	})

	// 初始化运行环境
	EnvInit(cfg)

	// 检查nodeagent.pid文件，判断是否已经有进程在运行中
	if checkRunning(cfg) {
		log.Fatal("Another agent is running.")
		return
	}

	// 处理守护进程模式
	if cfg.DaemonMode {
		logFile := filepath.Join(cfg.GaiaSecDir, "logs", pb.AgentType_NODE_AGENT.String(), cfg.NodeID, "agent.log")
		logDir := path.Dir(logFile)
		util.MkdirAll(logDir, 0777)
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

// JavaAgentStyleFormatter 自定义日志格式化器，与JavaAgent保持一致
type JavaAgentStyleFormatter struct {
	NodeID string
}

// Format 实现logrus.Formatter接口
func (f *JavaAgentStyleFormatter) Format(entry *log.Entry) ([]byte, error) {
	// 获取日志级别
	level := strings.ToUpper(entry.Level.String())

	// 构建消息内容
	message := entry.Message

	// 格式化时间
	timestamp := entry.Time.Format("2006-01-02 15:04:05")

	// 构建日志行，格式：时间 [AGENT_TYPE][LEVEL]nodeId message
	logLine := fmt.Sprintf("%s [NODE_AGENT][%s]%s %s\n", timestamp, level, f.NodeID, message)

	return []byte(logLine), nil
}

// 自定义日志格式化器，与JavaAgent保持一致
func init() {
	// 默认使用简单的文本格式，后续会在ParseMain中更新为自定义格式
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: true,
		DisableColors:    true,
	})
}

// EnvInit 初始化NodeAgent运行环境
// 创建必要的目录结构并设置环境变量，包括：
// - GaiaSec工作目录
// - 日志目录
// - 临时文件目录
// - Shell环境变量配置
func EnvInit(cfg *config.Config) {
	// 初始化各个目录
	// 创建gaiasec主目录，权限设置为777
	if err := util.MkdirAll(cfg.GaiaSecDir, 0777); err != nil {
		log.Fatalf("create gaiasec dir error: %v", err)
	}

	// 创建日志文件目录
	logDir := filepath.Join(cfg.GaiaSecDir, "logs/", pb.AgentType_NODE_AGENT.String(), "/", cfg.NodeID, "/")
	if err := util.MkdirAll(logDir, 0777); err != nil {
		log.Fatalf("create log dir error: %v", err)
	}

	// 创建临时文件目录
	tmpDir := filepath.Join(cfg.GaiaSecDir, "tmp")
	if err := util.MkdirAll(tmpDir, 0777); err != nil {
		log.Fatalf("create tmp dir error: %v", err)
	}

	// 初始化环境变量
	// 设置Shell相关的环境变量以优化终端体验
	envVars := map[string]string{
		"TMOUT":    "0",                                            // 禁用shell超时
		"HISTSIZE": "1000",                                         // 历史命令数量
		"HISTFILE": filepath.Join(cfg.GaiaSecDir, ".bash_history"), // 历史文件路径
		"PATH":     os.Getenv("PATH") + ":" + cfg.GaiaSecDir,       // 添加gaiasec bin目录到PATH
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

	log.Infof("EnvInit success: gaiasec_dir=%s", cfg.GaiaSecDir)
}

// checkRunning 检查是否已有NodeAgent实例在运行
// 通过读取PID文件并检查进程是否存在来判断
// 返回true表示已有实例在运行，false表示没有
func checkRunning(cfg *config.Config) bool {
	pidFile := filepath.Join(cfg.GaiaSecDir, "nodeagent.pid")
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
	pidFile := filepath.Join(cfg.GaiaSecDir, "nodeagent.pid")
	pid := os.Getpid()
	if err := ioutil.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0644); err != nil {
		log.Fatalf("create pid file error: %v", err)
	}
}
