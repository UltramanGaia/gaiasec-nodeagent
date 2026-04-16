package naserver

import (
	"crypto/tls"
	"gaiasec-nodeagent/pkg/config"
	"gaiasec-nodeagent/pkg/filesystem"
	"gaiasec-nodeagent/pkg/pb"
	"gaiasec-nodeagent/pkg/util"
	"io"
	"mime/multipart"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

const archiveTempDir = "/gaiasec/tmp"

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

func (na *NodeAgent) handleFsDownload(message *pb.Base) {
	log.Info("handle file system download file")
	request := &pb.FSDownloadFileRequest{}
	if err := proto.Unmarshal(message.Data, request); err != nil {
		log.Info("unmarshal error:", err)
		return
	}

	r, w := io.Pipe()
	m := multipart.NewWriter(w)
	defer r.Close()
	go func() {
		defer w.Close()
		defer m.Close()
		part, err := m.CreateFormFile("file", message.Session)
		if err != nil {
			log.Info("create form file error:", err)
			return
		}
		file, err := os.Open(request.Path)
		if err != nil {
			log.Info("open file error:", err)
			return
		}
		defer file.Close()
		if _, err = io.Copy(part, file); err != nil {
			log.Info("copy file error:", err)
		}
	}()

	cfg := config.GetInstance()

	// 创建自定义HTTP客户端，禁用TLS证书验证
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	protocol, host := util.ParseServerURL(cfg.Server)
	res, err := httpClient.Post(protocol+"://"+host+"/remote/filesystem/upload", m.FormDataContentType(), r)
	if err != nil {
		log.Info("post error:", err)
		return
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Info("read body error:", err)
		return
	}
	log.Info("upload file response:", string(body))
}

func (na *NodeAgent) handleFsArchiveCreate(message *pb.Base) {
	log.Info("handle file system archive create")
	request := &pb.FSArchiveCreateRequest{}
	if err := proto.Unmarshal(message.Data, request); err != nil {
		log.Info("unmarshal error:", err)
		return
	}

	if err := os.MkdirAll(archiveTempDir, 0o755); err != nil {
		_ = na.wsClient.SendMessage(&pb.FSArchiveCreateResponse{
			Result: "error",
			Error:  err.Error(),
		}, pb.MessageType_FS_ARCHIVE_CREATE_RESPONSE, na.NodeID, message.Source, message.Session)
		return
	}

	tempFile, err := os.CreateTemp(archiveTempDir, "gaiasec-archive-*.zip")
	if err != nil {
		_ = na.wsClient.SendMessage(&pb.FSArchiveCreateResponse{
			Result: "error",
			Error:  err.Error(),
		}, pb.MessageType_FS_ARCHIVE_CREATE_RESPONSE, na.NodeID, message.Source, message.Session)
		return
	}
	tempPath := tempFile.Name()
	tempFile.Close()

	log.Infof("create archive for path '%s' to '%s'", request.RootPath, tempPath)

	size, err := filesystem.CreateArchive(request.RootPath, request.Include, request.Exclude, tempPath)
	if err != nil {
		_ = os.Remove(tempPath)
		_ = na.wsClient.SendMessage(&pb.FSArchiveCreateResponse{
			Result: "error",
			Error:  err.Error(),
		}, pb.MessageType_FS_ARCHIVE_CREATE_RESPONSE, na.NodeID, message.Source, message.Session)
		return
	}
	log.Info("archive size:", size)

	if err := na.wsClient.SendMessage(&pb.FSArchiveCreateResponse{
		Result:      "ok",
		ArchivePath: tempPath,
		Size:        size,
	}, pb.MessageType_FS_ARCHIVE_CREATE_RESPONSE, na.NodeID, message.Source, message.Session); err != nil {
		log.Info("send archive create response error:", err)
	}
}
