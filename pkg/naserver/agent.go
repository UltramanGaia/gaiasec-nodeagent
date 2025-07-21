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
	"sothoth-nodeagent/pkg/system"
	"sothoth-nodeagent/pkg/udsserver"
	"sothoth-nodeagent/pkg/wsclient"
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

	wsclient  *wsclient.Client
	udsserver *udsserver.Server
	running   bool          // 运行状态标志
	stopChan  chan struct{} // 停止信号通道
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

	wsClient, err := wsclient.NewClient(serverURL, 5, 5)
	if err != nil {
		return nil, fmt.Errorf("创建WebSocket客户端失败: %v", err)
	}
	udsServer, err := udsserver.NewServer(wsClient)
	if err != nil {
		return nil, fmt.Errorf("创建UDS服务器失败: %v", err)
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
		wsclient:     wsClient,
		udsserver:    udsServer,
	}

	return agent, nil
}

// Start 启动Agent并维护连接
// 这是Agent的主运行循环，负责：
// - 建立与服务器的WebSocket连接
// - 处理连接断开后的自动重连
// - 维护连接状态直到Agent停止
//
// 返回：
//
//	error - 运行过程中的错误
func (na *NodeAgent) Start() error {
	log.Printf("启动Sothoth Node Agent v%s", na.AgentVersion)
	log.Printf("项目ID: %s", na.ProjectID)
	log.Printf("节点ID: %s", na.NodeID)
	log.Printf("主机名: %s", na.Hostname)
	log.Printf("IP地址: %s", na.IPAddress)
	log.Printf("Sothoth目录: %s", na.SothothDir)
	log.Printf("代理模式: %t", na.ProxyMode)
	log.Printf("连接到: %s", na.ServerURL)

	na.running = true

	go na.wsclient.Start()
	go na.udsserver.Start()

	return nil
}

// Stop 优雅地停止Agent
// 关闭所有连接和资源，包括：
// - 设置运行状态为false
// - 关闭停止信号通道
// - 关闭传统WebSocket连接
func (na *NodeAgent) Stop() {
	na.running = false

	na.udsserver.Stop()
	na.wsclient.Stop()

	log.Printf("Node Agent已停止")
}

//// handleMessage 处理传入的WebSocket消息
//// 根据消息类型分发到相应的处理函数
////
//// 参数：
////
////	msg - WebSocket消息对象
//func (na *NodeAgent) handleMessage(msg WebSocketMessage) {
//	switch msg.Type {
//	case "GET_PROCESSES":
//		na.handleGetProcesses(msg.RequestID)
//	case "EXECUTE_COMMAND":
//		if data, ok := msg.Data.(map[string]interface{}); ok {
//			if command, ok := data["command"].(string); ok {
//				na.handleExecuteCommand(msg.RequestID, command)
//			}
//		}
//	case "DEPLOY_PLUGIN":
//		na.handleDeployPlugin(msg)
//	default:
//		log.Printf("未知消息类型: %s", msg.Type)
//	}
//}

//// handleDeployPlugin 处理插件部署请求
//func (na *NodeAgent) handleDeployPlugin(msg WebSocketMessage) {
//	log.Printf("收到插件部署请求: %s", msg.RequestID)
//
//	data, ok := msg.Data.(map[string]interface{})
//	if !ok {
//		na.sendErrorResponse(msg.RequestID, "无效的插件部署数据")
//		return
//	}
//
//	// 解析部署参数
//	pluginName, _ := data["plugin_name"].(string)
//	pluginVersion, _ := data["plugin_version"].(string)
//	pluginURL, _ := data["plugin_url"].(string)
//	targetPID := int(data["target_pid"].(float64))
//	deploymentOptions, _ := data["deployment_options"].(map[string]interface{})
//
//	if pluginName == "" || pluginVersion == "" || pluginURL == "" {
//		na.sendErrorResponse(msg.RequestID, "缺少必需的插件部署参数")
//		return
//	}
//
//	log.Printf("部署插件: %s 版本: %s 到进程: %d", pluginName, pluginVersion, targetPID)
//
//	// 异步部署插件
//	go func() {
//		err := na.pluginManager.DeployPlugin(pluginName, pluginVersion, pluginURL, targetPID, deploymentOptions)
//		if err != nil {
//			log.Printf("插件部署失败: %v", err)
//		} else {
//			log.Printf("插件部署成功: %s", pluginName)
//		}
//	}()
//
//	// 立即返回部署开始响应
//	response := map[string]interface{}{
//		"success":     true,
//		"message":     "插件部署已开始",
//		"plugin_name": pluginName,
//	}
//
//	responseMsg := WebSocketMessage{
//		Type:      model.PLUGIN_DEPLOY_RESPONSE,
//		RequestID: msg.RequestID,
//		Data:      response,
//	}
//
//	na.sendMessage(responseMsg)
//}
