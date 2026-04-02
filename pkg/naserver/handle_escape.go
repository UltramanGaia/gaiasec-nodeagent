package naserver

import (
	"gaiasec-nodeagent/pkg/constant"
	"gaiasec-nodeagent/pkg/escape"
	"gaiasec-nodeagent/pkg/pb"
	log "github.com/sirupsen/logrus"
)

// handleContainerEscapeRequest 处理容器逃逸分析请求
func (na *NodeAgent) handleContainerEscapeRequest(message *pb.Base) {
	log.Info("handleContainerEscapeRequest start")

	// 执行容器逃逸分析
	log.Info("Calling AnalyzeContainerEscape...")
	results, err := escape.AnalyzeContainerEscape()
	log.Infof("AnalyzeContainerEscape returned, err=%v", err)
	if err != nil {
		log.Error("AnalyzeContainerEscape failed:", err)
		results = []*pb.ContainerEscapeInfo{}
	}

	log.Infof("Container escape analysis completed, found %d containers", len(results))

	// 通过 WebSocket 发送响应给请求方
	response := &pb.ContainerEscapeResponse{
		Containers: results,
	}

	err = na.wsClient.SendMessage(response, pb.MessageType_CONTAINER_ESCAPE_RESPONSE, message.Destination, message.Source, message.Session)
	if err != nil {
		log.Error("Send container escape response error:", err)
		return
	}

	log.Info("Container escape response sent to requester")

	// 通过 WebSocket 主动上报给 server
	err = na.wsClient.SendMessage(response, pb.MessageType_CONTAINER_ESCAPE_RESPONSE, constant.SERVER_ID, na.NodeID, message.Session)
	if err != nil {
		log.Error("Report container escape to server error:", err)
		return
	}

	log.Info("Container escape reported to server successfully")
}

// handlePrivilegeEscalationRequest 处理本地提权分析请求
func (na *NodeAgent) handlePrivilegeEscalationRequest(message *pb.Base) {
	log.Info("handlePrivilegeEscalationRequest")

	// 执行本地提权分析
	info, err := escape.AnalyzePrivilegeEscalation()
	if err != nil {
		log.Error("AnalyzePrivilegeEscalation failed:", err)
		info = &pb.PrivilegeEscalationInfo{}
	}

	log.Info("Privilege escalation analysis completed")

	// 通过 WebSocket 发送响应给请求方
	response := &pb.PrivilegeEscalationResponse{
		Info: info,
	}

	err = na.wsClient.SendMessage(response, pb.MessageType_PRIVILEGE_ESCALATION_RESPONSE, message.Destination, message.Source, message.Session)
	if err != nil {
		log.Error("Send privilege escalation response error:", err)
		return
	}

	log.Info("Privilege escalation response sent to requester")

	// 通过 WebSocket 主动上报给 server
	err = na.wsClient.SendMessage(response, pb.MessageType_PRIVILEGE_ESCALATION_RESPONSE, constant.SERVER_ID, na.NodeID, message.Session)
	if err != nil {
		log.Error("Report privilege escalation to server error:", err)
		return
	}

	log.Info("Privilege escalation reported to server successfully")
}

// handleK8SPrivilegeEscalationRequest 处理 K8s 提权分析请求
func (na *NodeAgent) handleK8SPrivilegeEscalationRequest(message *pb.Base) {
	log.Info("handleK8SPrivilegeEscalationRequest")

	// 执行 K8s 提权分析
	info, err := escape.AnalyzeK8SPrivilegeEscalation()
	if err != nil {
		log.Error("AnalyzeK8SPrivilegeEscalation failed:", err)
		info = &pb.K8SPrivilegeEscalationInfo{}
	}

	log.Info("K8s privilege escalation analysis completed")

	// 通过 WebSocket 发送响应给请求方
	response := &pb.K8SPrivilegeEscalationResponse{
		Info: info,
	}

	err = na.wsClient.SendMessage(response, pb.MessageType_K8S_PRIVILEGE_ESCALATION_RESPONSE, message.Destination, message.Source, message.Session)
	if err != nil {
		log.Error("Send K8s privilege escalation response error:", err)
		return
	}

	log.Info("K8s privilege escalation response sent to requester")

	// 通过 WebSocket 主动上报给 server
	err = na.wsClient.SendMessage(response, pb.MessageType_K8S_PRIVILEGE_ESCALATION_RESPONSE, constant.SERVER_ID, na.NodeID, message.Session)
	if err != nil {
		log.Error("Report K8s privilege escalation to server error:", err)
		return
	}

	log.Info("K8s privilege escalation reported to server successfully")
}