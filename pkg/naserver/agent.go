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
	log "github.com/sirupsen/logrus"
	"sothoth-nodeagent/pkg/config"
	"sothoth-nodeagent/pkg/constant"
	"sothoth-nodeagent/pkg/pb"
	"sothoth-nodeagent/pkg/process"
	"sothoth-nodeagent/pkg/proxy"
	"sothoth-nodeagent/pkg/system"
	"sothoth-nodeagent/pkg/udsserver"
	"sothoth-nodeagent/pkg/wsclient"
	"time"
)

// NodeAgent 代表主要的Agent结构体
// 包含Agent的所有配置信息和运行状态
type NodeAgent struct {
	ProjectID    string
	NodeID       string
	ServerURL    string
	SothothDir   string
	ProxyMode    bool
	Sock5Addr    string
	AutoHook     bool
	AgentVersion string
	Hostname     string
	IPAddress    []string

	wsClient    *wsclient.Client
	udsServer   *udsserver.Server
	proxyServer *proxy.Server // 代理

	running  bool          // 运行状态标志
	stopChan chan struct{} // 停止信号通道
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
func NewNodeAgent() (*NodeAgent, error) {
	cfg := config.GetInstance()

	serverURL := fmt.Sprintf("ws://%s/ws/agent?projectId=%s&connectId=%s", cfg.Server, cfg.ProjectID, cfg.NodeID)
	hostname, err := system.GetHostname()
	if err != nil {
		return nil, fmt.Errorf("获取主机名失败: %v", err)
	}

	ipAddress, err := system.GetLocalIps()
	if err != nil {
		return nil, fmt.Errorf("获取IP地址失败: %v", err)
	}

	wsClient, err := wsclient.NewClient(serverURL, 10, 10)
	if err != nil {
		return nil, fmt.Errorf("创建WebSocket客户端失败: %v", err)
	}
	udsServer, err := udsserver.NewServer(wsClient)
	if err != nil {
		return nil, fmt.Errorf("创建UDS服务器失败: %v", err)
	}

	proxyServer, err := proxy.NewServer(wsClient, cfg.Socks5Addr)
	if err != nil {
		return nil, fmt.Errorf("创建代理服务器失败: %v", err)
	}

	agent := &NodeAgent{
		ProjectID:    cfg.ProjectID,
		NodeID:       cfg.NodeID,
		ServerURL:    serverURL,
		SothothDir:   cfg.SothothDir,
		ProxyMode:    cfg.ProxyMode,
		Sock5Addr:    cfg.Socks5Addr,
		AutoHook:     cfg.AutoHook,
		AgentVersion: "1.0.0",
		Hostname:     hostname,
		IPAddress:    ipAddress,
		running:      false,
		stopChan:     make(chan struct{}),
		wsClient:     wsClient,
		udsServer:    udsServer,
		proxyServer:  proxyServer,
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
	log.Infof("Sothoth Node Agent v%s", na.AgentVersion)
	log.Infof("ProjectID: %s", na.ProjectID)
	log.Infof("NodeID: %s", na.NodeID)
	log.Infof("HostName: %s", na.Hostname)
	log.Infof("IP Addr: %s", na.IPAddress)
	log.Infof("Sothoth Dir: %s", na.SothothDir)
	log.Infof("ProxyMode: %t", na.ProxyMode)
	log.Infof("ConnectUrl: %s", na.ServerURL)

	na.running = true

	na.wsClient.Start()
	na.udsServer.Start()
	// todo

	go na.handleWsMessages()
	go na.handleUdsMessages()
	if na.Sock5Addr != "" {
		go na.handleProxyMessages()
	}

	if na.AutoHook {
		go na.monitorProcess()
	}

	return nil
}

func (na *NodeAgent) handleWsMessages() {
	// 连接建立成功，上报节点信息
	na.reportRegister()

	go na.heartbeatLoop()

	for {
		// 解析基础消息
		baseMessage := &pb.Base{}
		err := na.wsClient.ReadMessage(baseMessage)
		if err != nil {
			if !na.wsClient.Running {
				return
			}
			log.Error("read:", err)
			err = na.wsClient.Reconnect()
			if err != nil {
				log.Error("Reconnect failed, wait 5 mins")
				time.Sleep(5 * time.Minute)
				continue
			}
			na.reportRegister()
			continue
		}

		destination := baseMessage.Destination
		if na.NodeID == destination {
			log.Debug("receive message, handle it.")
			// 根据消息类型处理
				switch baseMessage.Type {
				case pb.MessageType_PROCESSES_REQUEST:
					go na.handleProcessRequest(baseMessage)
				case pb.MessageType_NETWORK_REQUEST:
					go na.handleNetworkRequest(baseMessage)
				case pb.MessageType_DEPLOY_PLUGIN_REQUEST:
					go na.handleDeployPluginRequest(baseMessage)
				case pb.MessageType_EXECUTE_COMMAND_REQUEST:
					go na.handleExecuteCommand(baseMessage)
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
				case pb.MessageType_FS_DOWNLOAD_REQUEST:
					go na.handleFsDownload(baseMessage)
				case pb.MessageType_PROXY_CLOSE: // closed by client
					go na.handleProxyClose(baseMessage)
				case pb.MessageType_PROXY_ESTABLISH: // establish
					go na.handleProxyEstablish(baseMessage)
				case pb.MessageType_PROXY_DATA_TO_SERVER:
					go na.handleProxyDataToServer(baseMessage)
				case pb.MessageType_PROXY_DATA_TO_CLIENT:
					go na.handleProxyDataToClient(baseMessage)
				case pb.MessageType_TERMINAL_CREATE_REQUEST:
					go na.handlePtyCreate(baseMessage)
				default:
					log.Error("UNKNOWN MESSAGE TYPE")
				}
		} else {
			log.Info("receive message, route it.")
			na.routeToAgent(baseMessage)
		}
	}
}

func (na *NodeAgent) handleUdsMessages() {
	na.udsServer.HandleAgentMessage()
}

func (na *NodeAgent) handleProxyMessages() {
	na.proxyServer.HandleSocks5Message()
}

// Stop 优雅地停止Agent
// 关闭所有连接和资源，包括：
// - 设置运行状态为false
// - 关闭停止信号通道
// - 关闭传统WebSocket连接
func (na *NodeAgent) Stop() {
	na.running = false

	na.udsServer.Stop()
	na.wsClient.Stop()

	log.Infof("Node Agent stop.")
}

// 转发给特定目标，从WsServer收到消息，通过uds转发给下面的 agent
func (na *NodeAgent) routeToAgent(message *pb.Base) {
	log.Info("route message to agent: " + message.Destination)
	na.udsServer.HandleMessage(message)
}

func (na *NodeAgent) monitorProcess() {
	for {
		if !na.running {
			return
		}

		processes, err := process.GetProcessList()
		if err != nil {
			log.Info("GetProcesses failed:", err)
		}
		response := &pb.ProcessesResponse{
			Processes: processes,
		}

		err = na.wsClient.SendMessage(response, pb.MessageType_PROCESSES_RESPONSE, na.NodeID, constant.SERVER_ID, constant.SESSIOND_ID_EMPTY)
		if err != nil {
			log.Info("Send error:", err)
		}

		time.Sleep(1 * time.Minute)
	}
}
