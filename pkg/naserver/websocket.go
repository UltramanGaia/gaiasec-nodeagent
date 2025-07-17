package naserver

import (
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketMessage represents the message structure for WebSocket communication
type WebSocketMessage struct {
	Type      string      `json:"type"`
	RequestID string      `json:"request_id,omitempty"`
	Data      interface{} `json:"data,omitempty"`
}


// connect establishes WebSocket connection to the server
func (a *NodeAgent) connect() error {
	u, err := url.Parse(a.ServerURL)
	if err != nil {
		return fmt.Errorf("invalid server URL: %v", err)
	}

	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}
	conn.SetReadLimit(0) // 0代表不限制读取

	a.conn = conn
	log.Println("Connected to Sothoth server")
	
	// Node registration is now handled automatically by the server
	// based on the projectId and nodeId in the connection URL
	log.Printf("Node connection established for Project: %s, Node: %s", a.ProjectID, a.NodeID)

	return nil
}

// handleConnection handles the WebSocket connection lifecycle
func (a *NodeAgent) handleConnection() {
	// Start heartbeat goroutine
	go a.heartbeatLoop()

	// Handle incoming messages
	for a.running {
		var msg WebSocketMessage
		err := a.conn.ReadJSON(&msg)
		if err != nil {
			if a.running {
				log.Printf("Read error: %v", err)
			}
			break
		}

		go a.handleMessage(msg)
	}
}

// heartbeatLoop sends periodic heartbeat messages
func (a *NodeAgent) heartbeatLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !a.running {
				return
			}

			heartbeat := WebSocketMessage{
				Type: "HEARTBEAT",
				Data: map[string]interface{}{},
			}

			if err := a.sendMessage(heartbeat); err != nil {
				log.Printf("Heartbeat error: %v", err)
				return
			}
		case <-a.stopChan:
			return
		}
	}
}

// sendMessage sends a WebSocket message
func (a *NodeAgent) sendMessage(msg WebSocketMessage) error {
	if a.conn == nil {
		return fmt.Errorf("connection not established")
	}
	return a.conn.WriteJSON(msg)
}
