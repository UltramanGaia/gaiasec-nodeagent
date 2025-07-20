package naserver

import (
	"log"
	"sothoth-nodeagent/pkg/system"
)

// WebSocket版本的进程列表处理器
func (a *NodeAgent) handleGetProcessesWS(msg WebSocketMessage) {
	a.handleGetProcesses(msg.RequestID)
}

// handleGetProcesses 处理获取进程列表的请求
// 调用系统接口获取当前运行的进程列表，并返回给服务器
//
// 参数：
//
//	requestID - 请求ID，用于关联响应
func (a *NodeAgent) handleGetProcesses(requestID string) {
	processes, err := system.GetProcessList()
	if err != nil {
		log.Printf("获取进程列表失败: %v", err)
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
		log.Printf("发送进程列表响应失败: %v", err)
	}
}
