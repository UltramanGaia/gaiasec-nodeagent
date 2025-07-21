package naserver

//
//import (
//	"sothoth-nodeagent/pkg/model"
//	"sothoth-nodeagent/pkg/terminal"
//	"sothoth-nodeagent/pkg/util"
//)
//
//// PTY相关处理方法
//func (na *NodeAgent) handlePtyCreate(msg WebSocketMessage) {
//	data, ok := msg.Data.(map[string]interface{})
//	if !ok {
//		na.sendErrorResponse(msg.RequestID, "Invalid data format")
//		return
//	}
//
//	cols, _ := data["cols"].(uint16)
//	rows, _ := data["rows"].(uint16)
//	sessionID, _ := data["session_id"].(string)
//
//	cmds := []string{"/bin/zsh", "/bin/bash", "/bin/dash", "/bin/sh"}
//	for _, cmd := range cmds {
//		if util.Exists(cmd) {
//
//			go terminal.CreateNewTerminalSocket(cols, rows, cmd, sessionID)
//
//			// 发送成功响应
//			response := WebSocketMessage{
//				Type:      model.PTY_CREATED,
//				RequestID: msg.RequestID,
//				Data: map[string]interface{}{
//					"sessionId": sessionID,
//				},
//			}
//			na.sendMessage(response)
//			return
//		}
//	}
//}
