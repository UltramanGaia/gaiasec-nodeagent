package naserver

//func (na *NodeAgent) handlePluginDeploy(msg WebSocketMessage) {
//	data, ok := msg.Data.(map[string]interface{})
//	if !ok {
//		na.sendErrorResponse(msg.RequestID, "Invalid data format")
//		return
//	}
//
//	pluginName, _ := data["plugin_name"].(string)
//	pluginVersion, _ := data["plugin_version"].(string)
//	agentId, _ := data["agent_id"].(string)
//	targetPid, _ := data["target_pid"].(string)
//	pid, _ := strconv.Atoi(targetPid)
//
//	// 验证必需参数
//	if pluginName == "" || pluginVersion == "" {
//		na.sendErrorResponse(msg.RequestID, "Missing required parameters: plugin_name or plugin_version")
//		return
//	}
//
//	na.pluginManager.DeployPlugin(pluginName, pluginVersion, agentId, pid, nil)
//
//	// 构建成功响应
//	response := map[string]interface{}{
//		"success":        true,
//		"message":        "Plugin deployed successfully",
//		"plugin_name":    pluginName,
//		"plugin_version": pluginVersion,
//	}
//
//	responseMsg := WebSocketMessage{
//		Type:      model.PLUGIN_DEPLOY_RESPONSE,
//		RequestID: msg.RequestID,
//		Data:      response,
//	}
//
//	err := na.sendMessage(responseMsg)
//	if err != nil {
//		log.Printf("发送插件部署响应失败: %v", err)
//	} else {
//		log.Printf("插件部署成功响应已发送: %s", pluginName)
//	}
//}
