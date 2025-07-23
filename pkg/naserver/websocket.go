package naserver

// WebSocketMessage represents the message structure for WebSocket communication
type WebSocketMessage struct {
	Type      string      `json:"type"`
	RequestID string      `json:"request_id,omitempty"`
	Data      interface{} `json:"data,omitempty"`
}

//// connect establishes WebSocket connection to the server
//func (na *NodeAgent) connect() error {
//	u, err := url.Parse(na.ServerURL)
//	if err != nil {
//		return fmt.Errorf("invalid server URL: %v", err)
//	}
//
//	dialer := websocket.DefaultDialer
//	conn, _, err := dialer.Dial(u.String(), nil)
//	if err != nil {
//		return fmt.Errorf("failed to connect: %v", err)
//	}
//	conn.SetReadLimit(0) // 0代表不限制读取
//
//	na.conn = conn
//	log.Info("Connected to Sothoth server")
//
//	// Node registration is now handled automatically by the server
//	// based on the projectId and nodeId in the connection URL
//	log.Printf("Node connection established for Project: %s, Node: %s", na.ProjectID, na.NodeID)
//
//	return nil
//}

//// handleConnection handles the WebSocket connection lifecycle
//func (na *NodeAgent) handleConnection() {
//	// Start heartbeat goroutine
//	go na.heartbeatLoop()
//
//	// handler agent message
//	go na.handleAgentMessage()
//
//	// Handle incoming messages
//	for na.running {
//		var msg WebSocketMessage
//		err := na.conn.ReadJSON(&msg)
//		if err != nil {
//			if na.running {
//				log.Printf("Read error: %v", err)
//			}
//			break
//		}
//
//		go na.processMessage(msg)
//	}
//}

//// heartbeatLoop sends periodic heartbeat messages
//func (na *NodeAgent) heartbeatLoop() {
//	ticker := time.NewTicker(30 * time.Second)
//	defer ticker.Stop()
//
//	for {
//		select {
//		case <-ticker.C:
//			if !na.running {
//				return
//			}
//
//			heartbeat := WebSocketMessage{
//				Type: model.HEARTBEAT,
//				Data: map[string]interface{}{},
//			}
//
//			if err := na.sendMessage(heartbeat); err != nil {
//				log.Printf("Heartbeat error: %v", err)
//				return
//			}
//		case <-na.stopChan:
//			return
//		}
//	}
//}

//// processMessage processes incoming WebSocket messages (renamed to avoid conflict)
//func (na *NodeAgent) processMessage(msg WebSocketMessage) {
//	switch msg.Type {
//	case model.EXECUTE_COMMAND:
//		na.handleExecuteCommandWS(msg)
//	case model.GET_PROCESSES:
//		na.handleGetProcessesWS(msg)
//	case model.PTY_CREATE:
//		na.handlePtyCreate(msg)
//	case model.FS_LIST_DIR:
//		na.handleFsListDir(msg)
//	case model.FS_READ_FILE:
//		na.handleFsReadFile(msg)
//	case model.FS_WRITE_FILE:
//		na.handleFsWriteFile(msg)
//	case model.FS_CREATE_FILE:
//		na.handleFsCreateFile(msg)
//	case model.FS_CREATE_DIR:
//		na.handleFsCreateDir(msg)
//	case model.FS_DELETE:
//		na.handleFsDelete(msg)
//	case model.FS_RENAME:
//		na.handleFsRename(msg)
//	case model.DEPLOY_PLUGIN:
//		na.handlePluginDeploy(msg)
//	default:
//		log.Printf("Unknown message type: %s", msg.Type)
//	}
//}

//// sendMessage sends a WebSocket message
//func (na *NodeAgent) sendMessage(msg WebSocketMessage) error {
//	if na.conn == nil {
//		return fmt.Errorf("connection not established")
//	}
//	na.mu.Lock()
//	defer na.mu.Unlock()
//	return na.conn.WriteJSON(msg)
//}

//// emit
//func (na *NodeAgent) emit(message string, args ...interface{}) error {
//	args = append([]interface{}{message}, args...)
//	response := WebSocketMessage{
//		Type:      model.EVENT,
//		RequestID: "",
//		Data:      args,
//	}
//	return na.sendMessage(response)
//}

//// sendErrorResponse 发送错误响应
//func (na *NodeAgent) sendErrorResponse(requestID, errorMsg string) {
//	response := map[string]interface{}{
//		"success": false,
//		"error":   errorMsg,
//	}
//
//	responseMsg := WebSocketMessage{
//		Type:      model.ERROR_RESPONSE,
//		RequestID: requestID,
//		Data:      response,
//	}
//
//	na.sendMessage(responseMsg)
//}
