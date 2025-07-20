package naserver

import "sothoth-nodeagent/pkg/filesystem"

// 文件系统相关处理方法
func (a *NodeAgent) handleFsListDir(msg WebSocketMessage) {
	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		a.sendErrorResponse(msg.RequestID, "Invalid data format")
		return
	}

	path, _ := data["path"].(string)
	if path == "" {
		path = "/"
	}

	files, err := filesystem.ListDirectory(path)
	if err != nil {
		a.sendErrorResponse(msg.RequestID, err.Error())
		return
	}

	response := WebSocketMessage{
		Type:      "FS_RESPONSE",
		RequestID: msg.RequestID,
		Data: map[string]interface{}{
			"files": files,
		},
	}
	a.sendMessage(response)
}

func (a *NodeAgent) handleFsReadFile(msg WebSocketMessage) {
	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		a.sendErrorResponse(msg.RequestID, "Invalid data format")
		return
	}

	path, _ := data["path"].(string)

	content, err := filesystem.ReadFile(path)
	if err != nil {
		a.sendErrorResponse(msg.RequestID, err.Error())
		return
	}

	response := WebSocketMessage{
		Type:      "FS_RESPONSE",
		RequestID: msg.RequestID,
		Data: map[string]interface{}{
			"content":      content.Content,
			"encoding":     content.Encoding,
			"size":         content.Size,
			"lastModified": content.LastModified,
			"isBinary":     content.IsBinary,
		},
	}
	a.sendMessage(response)
}

func (a *NodeAgent) handleFsWriteFile(msg WebSocketMessage) {
	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		a.sendErrorResponse(msg.RequestID, "Invalid data format")
		return
	}

	path, _ := data["path"].(string)
	content, _ := data["content"].(string)

	err := filesystem.WriteFile(path, content)
	if err != nil {
		a.sendErrorResponse(msg.RequestID, err.Error())
		return
	}

	response := WebSocketMessage{
		Type:      "FS_RESPONSE",
		RequestID: msg.RequestID,
		Data:      map[string]interface{}{"success": true},
	}
	a.sendMessage(response)
}

func (a *NodeAgent) handleFsCreateFile(msg WebSocketMessage) {
	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		a.sendErrorResponse(msg.RequestID, "Invalid data format")
		return
	}

	path, _ := data["path"].(string)

	err := filesystem.CreateFile(path)
	if err != nil {
		a.sendErrorResponse(msg.RequestID, err.Error())
		return
	}

	response := WebSocketMessage{
		Type:      "FS_RESPONSE",
		RequestID: msg.RequestID,
		Data:      map[string]interface{}{"success": true},
	}
	a.sendMessage(response)
}

func (a *NodeAgent) handleFsCreateDir(msg WebSocketMessage) {
	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		a.sendErrorResponse(msg.RequestID, "Invalid data format")
		return
	}

	path, _ := data["path"].(string)

	err := filesystem.CreateDirectory(path)
	if err != nil {
		a.sendErrorResponse(msg.RequestID, err.Error())
		return
	}

	response := WebSocketMessage{
		Type:      "FS_RESPONSE",
		RequestID: msg.RequestID,
		Data:      map[string]interface{}{"success": true},
	}
	a.sendMessage(response)
}

func (a *NodeAgent) handleFsDelete(msg WebSocketMessage) {
	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		a.sendErrorResponse(msg.RequestID, "Invalid data format")
		return
	}

	path, _ := data["path"].(string)

	err := filesystem.Delete(path)
	if err != nil {
		a.sendErrorResponse(msg.RequestID, err.Error())
		return
	}

	response := WebSocketMessage{
		Type:      "FS_RESPONSE",
		RequestID: msg.RequestID,
		Data:      map[string]interface{}{"success": true},
	}
	a.sendMessage(response)
}

func (a *NodeAgent) handleFsRename(msg WebSocketMessage) {
	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		a.sendErrorResponse(msg.RequestID, "Invalid data format")
		return
	}

	oldPath, _ := data["oldPath"].(string)
	newPath, _ := data["newPath"].(string)

	err := filesystem.Rename(oldPath, newPath)
	if err != nil {
		a.sendErrorResponse(msg.RequestID, err.Error())
		return
	}

	response := WebSocketMessage{
		Type:      "FS_RESPONSE",
		RequestID: msg.RequestID,
		Data:      map[string]interface{}{"success": true},
	}
	a.sendMessage(response)
}
