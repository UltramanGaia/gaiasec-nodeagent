package naserver

import (
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
	"sothoth-nodeagent/pkg/filesystem"
	"sothoth-nodeagent/pkg/pb"
)

// 文件系统相关处理方法
func (na *NodeAgent) handleFsListDir(message *pb.Base) {
	log.Info("handle file system list directory")
	request := &pb.FSListDirRequest{}
	if err := proto.Unmarshal(message.Data, request); err != nil {
		log.Info("unmarshal error:", err)
	}

	files, err := filesystem.ListDirectory(request.Path)
	if err != nil {
		log.Info("file system list directory error:", err)
	}
	if err != nil {
		log.Info("ListDirectory failed:", err)
	}
	response := &pb.FSListDirResponse{
		Path:   request.Path,
		Files:  files,
		Result: "ok",
	}

	err = na.wsClient.SendMessage(response, pb.MessageType_FS_LIST_DIR_RESPONSE, na.NodeID, message.Source, message.Session)
	if err != nil {
		log.Info("send resp error:", err)
		return
	}
}

func (na *NodeAgent) handleFsReadFile(message *pb.Base) {
	log.Info("handle file system read file")
	request := &pb.FSReadFileRequest{}
	if err := proto.Unmarshal(message.Data, request); err != nil {
		log.Info("unmarshal error:", err)
		return
	}
	response, err := filesystem.ReadFile(request.Path)
	if err != nil {
		log.Info("ReadFile failed:", err)
	}
	err = na.wsClient.SendMessage(response, pb.MessageType_FS_READ_FILE_RESPONSE, na.NodeID, message.Source, message.Session)
	if err != nil {
		log.Info("send resp error:", err)
		return
	}
}

func (na *NodeAgent) handleFsWriteFile(message *pb.Base) {
	log.Info("handle file system write file")
	request := &pb.FSWriteFileRequest{}
	if err := proto.Unmarshal(message.Data, request); err != nil {
		log.Info("unmarshal error:", err)
		return
	}

	err := filesystem.WriteFile(request.Path, request.Content)
	if err != nil {
		log.Info("WriteFile failed:", err)
	}
	response := &pb.FSWriteFileResponse{
		Result: "ok",
	}

	err = na.wsClient.SendMessage(response, pb.MessageType_FS_WRITE_FILE_RESPONSE, na.NodeID, message.Source, message.Session)
	if err != nil {
		log.Info("send resp error:", err)
		return
	}

}

func (na *NodeAgent) handleFsCreateFile(message *pb.Base) {
	log.Info("handle file system create file")
	request := &pb.FSCreateFileRequest{}
	if err := proto.Unmarshal(message.Data, request); err != nil {
		log.Info("unmarshal error:", err)
		return
	}

	err := filesystem.CreateFile(request.Path)
	if err != nil {
		log.Info("Create failed:", err)
	}
	response := &pb.FSCreateFileResponse{
		Result: "ok",
	}

	err = na.wsClient.SendMessage(response, pb.MessageType_FS_CREATE_FILE_RESPONSE, na.NodeID, message.Source, message.Session)
	if err != nil {
		log.Info("send resp error:", err)
		return
	}
}

func (na *NodeAgent) handleFsCreateDir(message *pb.Base) {

	log.Info("handle file system create directory")
	request := &pb.FSCreateDirRequest{}
	if err := proto.Unmarshal(message.Data, request); err != nil {
		log.Info("unmarshal error:", err)
		return
	}

	err := filesystem.CreateDirectory(request.Path)
	if err != nil {
		log.Info("Create failed:", err)
	}
	response := &pb.FSCreateFileResponse{
		Result: "ok",
	}

	err = na.wsClient.SendMessage(response, pb.MessageType_FS_CREATE_DIR_RESPONSE, na.NodeID, message.Source, message.Session)
	if err != nil {
		log.Info("send resp error:", err)
		return
	}
}

func (na *NodeAgent) handleFsDelete(message *pb.Base) {
	log.Info("handle file system create directory")
	request := &pb.FSDeleteFileRequest{}
	if err := proto.Unmarshal(message.Data, request); err != nil {
		log.Info("unmarshal error:", err)
		return
	}

	err := filesystem.Delete(request.Path)
	if err != nil {
		log.Info("Create failed:", err)
	}
	response := &pb.FSDeleteFileResponse{
		Result: "ok",
	}

	err = na.wsClient.SendMessage(response, pb.MessageType_FS_DELETE_RESPONSE, na.NodeID, message.Source, message.Session)
	if err != nil {
		log.Info("send resp error:", err)
		return
	}
}

func (na *NodeAgent) handleFsRename(message *pb.Base) {
	log.Info("handle file system rename")
	request := &pb.FSRenameFileRequest{}
	if err := proto.Unmarshal(message.Data, request); err != nil {
		log.Info("unmarshal error:", err)
		return
	}

	err := filesystem.Rename(request.OldPath, request.NewPath)
	if err != nil {
		log.Info("Create failed:", err)
	}
	response := &pb.FSRenameFileResponse{
		Result: "ok",
	}

	err = na.wsClient.SendMessage(response, pb.MessageType_FS_RENAME_RESPONSE, na.NodeID, message.Source, message.Session)
	if err != nil {
		log.Info("send resp error:", err)
		return
	}

}
