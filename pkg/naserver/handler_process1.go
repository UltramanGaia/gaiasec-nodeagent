package naserver

//
//import (
//	"log"
//	"sothoth-nodeagent/pkg/model"
//	"sothoth-nodeagent/pkg/process"
//)
//
//// WebSocket版本的进程列表处理器
//func (na *NodeAgent) handleGetProcessesWS(msg WebSocketMessage) {
//	na.handleGetProcesses(msg.RequestID)
//}
//
//// handleGetProcesses 处理获取进程列表的请求
//// 调用系统接口获取当前运行的进程列表，并返回给服务器
////
//// 参数：
////
////	requestID - 请求ID，用于关联响应
//func (na *NodeAgent) handleGetProcesses(requestID string) {
//	processes, err := process.GetProcessList()
//	if err != nil {
//		log.Printf("获取进程列表失败: %v", err)
//		return
//	}
//
//	response := WebSocketMessage{
//		Type:      model.PROCESSES_RESPONSE,
//		RequestID: requestID,
//		Data: map[string]interface{}{
//			"processes": processes,
//		},
//	}
//
//	if err := na.sendMessage(response); err != nil {
//		log.Printf("发送进程列表响应失败: %v", err)
//	}
//}
