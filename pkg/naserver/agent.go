package naserver

import (
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"sothoth-nodeagent/pkg/system"
)

// NodeAgent represents the main agent structure
type NodeAgent struct {
	ProjectID    string `json:"project_id"`
	NodeID       string `json:"node_id"`
	ServerURL    string `json:"server_url"`
	SothothDir   string `json:"sothoth_dir"`
	ProxyMode    bool   `json:"proxy_mode"`
	AgentVersion string `json:"agent_version"`
	Hostname     string `json:"hostname"`
	IPAddress    string `json:"ip_address"`

	conn     *websocket.Conn
	running  bool
	stopChan chan struct{}
}

// NewNodeAgent creates a new NodeAgent instance
func NewNodeAgent(projectID, nodeID, server, sothothDir string, proxyMode bool) (*NodeAgent, error) {
	serverURL := fmt.Sprintf("ws://%s/ws/nodeagent?projectId=%s&nodeId=%s", server, projectID, nodeID)
	hostname, err := system.GetHostname()
	if err != nil {
		return nil, fmt.Errorf("failed to get hostname: %v", err)
	}

	ipAddress, err := system.GetLocalIP()
	if err != nil {
		return nil, fmt.Errorf("failed to get IP address: %v", err)
	}

	return &NodeAgent{
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
	}, nil
}

// Run starts the agent and maintains connection
func (a *NodeAgent) Run() error {
	log.Printf("Starting Sothoth Node Agent v%s", a.AgentVersion)
	log.Printf("Project ID: %s", a.ProjectID)
	log.Printf("Node ID: %s", a.NodeID)
	log.Printf("Hostname: %s", a.Hostname)
	log.Printf("IP Address: %s", a.IPAddress)
	log.Printf("Sothoth Dir: %s", a.SothothDir)
	log.Printf("Proxy Mode: %t", a.ProxyMode)
	log.Printf("Connecting to: %s", a.ServerURL)

	a.running = true

	for a.running {
		if err := a.connect(); err != nil {
			log.Printf("Connection failed: %v", err)
			if a.running {
				log.Println("Reconnecting in 5 seconds...")
				time.Sleep(5 * time.Second)
			}
			continue
		}

		// Connection established, start message handling
		a.handleConnection()

		if a.running {
			log.Println("Connection lost, reconnecting in 5 seconds...")
			time.Sleep(5 * time.Second)
		}
	}

	return nil
}

// Stop gracefully stops the agent
func (a *NodeAgent) Stop() {
	a.running = false
	close(a.stopChan)
	if a.conn != nil {
		a.conn.Close()
	}
}

// handleMessage processes incoming WebSocket messages
func (a *NodeAgent) handleMessage(msg WebSocketMessage) {
	switch msg.Type {
	case "REGISTRATION_CONFIRMED":
		a.handleRegistrationConfirmed(msg)
	case "GET_PROCESSES":
		a.handleGetProcesses(msg.RequestID)
	case "EXECUTE_COMMAND":
		if data, ok := msg.Data.(map[string]interface{}); ok {
			if command, ok := data["command"].(string); ok {
				a.handleExecuteCommand(msg.RequestID, command)
			}
		}
	default:
		log.Printf("Unknown message type: %s", msg.Type)
	}
}

// handleGetProcesses handles process list requests
func (a *NodeAgent) handleGetProcesses(requestID string) {
	processes, err := system.GetProcessList()
	if err != nil {
		log.Printf("Failed to get processes: %v", err)
		return
	}

	response := WebSocketMessage{
		Type:      "PROCESSES_RESPONSE",
		RequestID: requestID,
		Data: map[string]interface{}{
			"processes": processes,
		},
	}

	if err := a.sendMessage(response); err != nil {
		log.Printf("Failed to send processes response: %v", err)
	}
}

// handleRegistrationConfirmed handles registration confirmation from server
func (a *NodeAgent) handleRegistrationConfirmed(msg WebSocketMessage) {
	if data, ok := msg.Data.(map[string]interface{}); ok {
		if nodeID, ok := data["node_id"].(string); ok {
			log.Printf("Registration confirmed. Server assigned node ID: %s", nodeID)
		}
		if status, ok := data["status"].(string); ok {
			log.Printf("Registration status: %s", status)
		}
	} else {
		log.Println("Registration confirmed by server")
	}
}

// handleExecuteCommand handles command execution requests
func (a *NodeAgent) handleExecuteCommand(requestID, command string) {
	result, err := system.ExecuteCommand(command)
	if err != nil {
		log.Printf("Failed to execute command: %v", err)
		result = &system.CommandResult{
			ExitCode:      -1,
			Stdout:        "",
			Stderr:        err.Error(),
			ExecutionTime: 0,
		}
	}

	response := WebSocketMessage{
		Type:      "COMMAND_RESULT",
		RequestID: requestID,
		Data:      result,
	}

	if err := a.sendMessage(response); err != nil {
		log.Printf("Failed to send command result: %v", err)
	}
}
