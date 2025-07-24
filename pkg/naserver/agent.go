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
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
	"sothoth-nodeagent/pkg/pb"
	"sothoth-nodeagent/pkg/system"
	"sothoth-nodeagent/pkg/udsserver"
	"sothoth-nodeagent/pkg/wsclient"
)

// NodeAgent 代表主要的Agent结构体
// 包含Agent的所有配置信息和运行状态
type NodeAgent struct {
	ProjectID    string
	NodeID       string
	ServerURL    string
	SothothDir   string
	ProxyMode    bool
	AgentVersion string
	Hostname     string
	IPAddress    []string

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
	serverURL := fmt.Sprintf("ws://%s/ws/agent?projectId=%s&connectId=%s", server, projectID, nodeID)
	hostname, err := system.GetHostname()
	if err != nil {
		return nil, fmt.Errorf("获取主机名失败: %v", err)
	}

	ipAddress, err := system.GetLocalIps()
	if err != nil {
		return nil, fmt.Errorf("获取IP地址失败: %v", err)
	}

	wsClient, err := wsclient.NewClient(serverURL, 10, 30)
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

	na.wsclient.Start()
	na.udsserver.Start()

	go na.handleWsMessages()
	go na.handleUdsMessages()

	return nil
}

func (na *NodeAgent) handleWsMessages() {
	// 连接建立成功，上报节点信息
	na.reportRegister()

	go na.heartbeatLoop()

	for {
		messageType, message, err := na.wsclient.ReadMessage()
		if err != nil {
			if !na.wsclient.Running {
				return

			}
			log.Error("read:", err)
			err = na.wsclient.Reconnect()
			if err != nil {
				log.Error("Reconnect failed")
				return
			}
			na.reportRegister()
			continue
		}

		if messageType == websocket.BinaryMessage {
			// 解析基础消息
			baseMessage := &pb.Base{}
			if err := proto.Unmarshal(message, baseMessage); err != nil {
				log.Info("解析基础消息失败:", err)
				continue
			}

			destination := baseMessage.Destination
			if na.NodeID == destination {
				log.Info("收到来自服务器的消息")
				// 根据消息类型处理
				switch baseMessage.Type {
				case pb.MessageType_PROCESSES_REQUEST:
					go na.handleProcessRequest(baseMessage)
				case pb.MessageType_DEPLOY_PLUGIN_REQUEST:
					go na.handleDeployPluginRequest(baseMessage)
				case pb.MessageType_FS_LIST_DIR_REQUEST:
					go na.handleFsListDir(baseMessage)
				case pb.MessageType_FS_READ_FILE_REQUEST:
					go na.handleFsReadFile(baseMessage)
				case pb.MessageType_FS_WRITE_FILE_REQUEST:
					go na.handleFsWriteFile(baseMessage)
				case pb.MessageType_FS_CREATE_FILE_REQUEST:
					go na.handleFsCreateFile(baseMessage)
				case pb.MessageType_FS_CREATE_DIR_REQUEST:
					go na.handleFsCreateDir(baseMessage)
				case pb.MessageType_FS_DELETE_REQUEST:
					go na.handleFsDelete(baseMessage)
				case pb.MessageType_FS_RENAME_REQUEST:
					go na.handleFsRename(baseMessage)
				default:
					log.Info("未知消息类型")
				}
			} else {
				na.routeToAgent(baseMessage)
			}
		}
	}
}

func (na *NodeAgent) handleUdsMessages() {
	na.udsserver.HandleAgentMessage()
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

// 转发给特定目标，从WsServer收到消息，通过uds转发给下面的 agent
func (na *NodeAgent) routeToAgent(message *pb.Base) {
	log.Info("route message to agent: " + message.Destination)
	na.udsserver.HandleMessage(message)
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
