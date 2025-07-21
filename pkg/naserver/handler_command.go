package naserver

//// WebSocket版本的命令处理器
//func (na *NodeAgent) handleExecuteCommandWS(msg WebSocketMessage) {
//	data, ok := msg.Data.(map[string]interface{})
//	if !ok {
//		na.sendErrorResponse(msg.RequestID, "Invalid data format")
//		return
//	}
//
//	command, ok := data["command"].(string)
//	if !ok {
//		na.sendErrorResponse(msg.RequestID, "Missing command")
//		return
//	}
//
//	na.handleExecuteCommand(msg.RequestID, command)
//}

//// handleExecuteCommand 处理命令执行请求
//// 执行指定的系统命令并返回执行结果
////
//// 参数：
////
////	requestID - 请求ID，用于关联响应
////	command - 要执行的命令
//func (na *NodeAgent) handleExecuteCommand(requestID, command string) {
//	result, err := system.ExecuteCommand(command)
//	if err != nil {
//		log.Printf("执行命令失败: %v", err)
//		result = &system.CommandResult{
//			ExitCode:      -1,
//			Stdout:        "",
//			Stderr:        err.Error(),
//			ExecutionTime: 0,
//		}
//	}
//
//	response := WebSocketMessage{
//		Type:      model.COMMAND_RESULT,
//		RequestID: requestID,
//		Data:      result,
//	}
//
//	if err := na.sendMessage(response); err != nil {
//		log.Printf("发送命令执行结果失败: %v", err)
//	}
//}
