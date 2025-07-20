// Package naserver 提供NodeAgent的核心功能实现
//
// 该包包含NodeAgent的主要结构和方法，负责：
// - 与Sothoth服务器建立和维护WebSocket连接
// - 处理来自服务器的各种请求（命令执行、进程查询等）
// - 管理终端会话和文件系统操作
// - 提供独立连接管理器支持多会话架构
package naserver

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"sothoth-nodeagent/pkg/system"
)

// NodeAgent 代表主要的Agent结构体
// 包含Agent的所有配置信息和运行状态
type NodeAgent struct {
	ProjectID    string `json:"project_id"`    // 所属项目ID
	NodeID       string `json:"node_id"`       // 节点唯一标识
	ServerURL    string `json:"server_url"`    // 服务器WebSocket URL
	SothothDir   string `json:"sothoth_dir"`   // Sothoth工作目录
	ProxyMode    bool   `json:"proxy_mode"`    // 是否启用代理模式
	AgentVersion string `json:"agent_version"` // Agent版本号
	Hostname     string `json:"hostname"`      // 主机名
	IPAddress    string `json:"ip_address"`    // IP地址

	conn     *websocket.Conn // WebSocket连接
	mu       sync.Mutex      // 保护写操作的互斥锁
	running  bool            // 运行状态标志
	stopChan chan struct{}   // 停止信号通道
}

// NewNodeAgent 创建一个新的NodeAgent实例
// 初始化Agent的基本配置，获取系统信息（主机名、IP地址），
// 并设置独立连接管理器
//
// 参数：
//
//	projectID - 项目ID
//	nodeID - 节点ID
//	server - 服务器地址
//	sothothDir - Sothoth工作目录
//	proxyMode - 是否启用代理模式
//
// 返回：
//
//	*NodeAgent - 创建的Agent实例
//	error - 创建过程中的错误
func NewNodeAgent(projectID, nodeID, server, sothothDir string, proxyMode bool) (*NodeAgent, error) {
	serverURL := fmt.Sprintf("ws://%s/ws/nodeagent?projectId=%s&nodeId=%s", server, projectID, nodeID)
	hostname, err := system.GetHostname()
	if err != nil {
		return nil, fmt.Errorf("获取主机名失败: %v", err)
	}

	ipAddress, err := system.GetLocalIP()
	if err != nil {
		return nil, fmt.Errorf("获取IP地址失败: %v", err)
	}

	agent := &NodeAgent{
		ProjectID:    projectID,
		NodeID:       nodeID,
		ServerURL:    serverURL,
		SothothDir:   sothothDir,
		ProxyMode:    proxyMode,
		AgentVersion: "1.0.0",
		Hostname:     hostname,
		IPAddress:    ipAddress,
		running:      false,
		stopChan:     make(chan struct{}),
	}

	return agent, nil
}

// Run 启动Agent并维护连接
// 这是Agent的主运行循环，负责：
// - 建立与服务器的WebSocket连接
// - 处理连接断开后的自动重连
// - 维护连接状态直到Agent停止
//
// 返回：
//
//	error - 运行过程中的错误
func (a *NodeAgent) Run() error {
	log.Printf("启动Sothoth Node Agent v%s", a.AgentVersion)
	log.Printf("项目ID: %s", a.ProjectID)
	log.Printf("节点ID: %s", a.NodeID)
	log.Printf("主机名: %s", a.Hostname)
	log.Printf("IP地址: %s", a.IPAddress)
	log.Printf("Sothoth目录: %s", a.SothothDir)
	log.Printf("代理模式: %t", a.ProxyMode)
	log.Printf("连接到: %s", a.ServerURL)

	a.running = true

	for a.running {
		if err := a.connect(); err != nil {
			log.Printf("连接失败: %v", err)
			if a.running {
				log.Println("5秒后重新连接...")
				time.Sleep(5 * time.Second)
			}
			continue
		}

		// 连接建立成功，上报节点信息
		a.reportNodeInfo()

		// 开始处理消息
		a.handleConnection()

		if a.running {
			log.Println("连接丢失，5秒后重新连接...")
			time.Sleep(5 * time.Second)
		}
	}

	return nil
}

// Stop 优雅地停止Agent
// 关闭所有连接和资源，包括：
// - 设置运行状态为false
// - 关闭停止信号通道
// - 关闭传统WebSocket连接
func (a *NodeAgent) Stop() {
	a.running = false
	close(a.stopChan)

	// 关闭传统连接
	if a.conn != nil {
		a.conn.Close()
	}

	log.Printf("Node Agent已停止")
}

// handleMessage 处理传入的WebSocket消息
// 根据消息类型分发到相应的处理函数
//
// 参数：
//
//	msg - WebSocket消息对象
func (a *NodeAgent) handleMessage(msg WebSocketMessage) {
	switch msg.Type {
	case "GET_PROCESSES":
		a.handleGetProcesses(msg.RequestID)
	case "EXECUTE_COMMAND":
		if data, ok := msg.Data.(map[string]interface{}); ok {
			if command, ok := data["command"].(string); ok {
				a.handleExecuteCommand(msg.RequestID, command)
			}
		}
	default:
		log.Printf("未知消息类型: %s", msg.Type)
	}
}

func (a *NodeAgent) reportNodeInfo() {
	// 构建节点信息数据
	nodeInfo := map[string]interface{}{
		"project_id":     a.ProjectID,
		"node_id":        a.NodeID,
		"hostname":       a.Hostname,
		"ip_address":     a.IPAddress,
		"agent_version":  a.AgentVersion,
		"sothoth_dir":    a.SothothDir,
		"proxy_mode":     a.ProxyMode,
	}

	// 创建节点信息上报消息
	msg := WebSocketMessage{
		Type: "NODE_INFO_REPORT",
		Data: nodeInfo,
	}

	// 发送节点信息到服务器
	if err := a.sendMessage(msg); err != nil {
		log.Printf("节点信息上报失败: %v", err)
	} else {
		log.Printf("节点信息上报成功 - 主机名: %s, IP: %s, 版本: %s", 
			a.Hostname, a.IPAddress, a.AgentVersion)
	}
}
