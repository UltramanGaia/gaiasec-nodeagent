package naserver

import (
	"gaiasec-nodeagent/pkg/container"
	"gaiasec-nodeagent/pkg/pb"
	log "github.com/sirupsen/logrus"
)

// handleContainerRequest 处理容器信息请求
func (na *NodeAgent) handleContainerRequest(message *pb.Base) {
	log.Info("handleContainerRequest")

	// 获取容器列表
	containers, err := container.GetContainerList()
	if err != nil {
		log.Error("GetContainerList failed:", err)
		containers = []*pb.Container{}
	}

	log.Infof("Found %d containers", len(containers))

	// 构造响应
	response := &pb.ContainerResponse{
		Containers: containers,
	}

	// 发送响应
	err = na.wsClient.SendMessage(response, pb.MessageType_CONTAINER_RESPONSE, message.Destination, message.Source, message.Session)
	if err != nil {
		log.Error("Send container response error:", err)
		return
	}
}
