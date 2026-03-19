package naserver

import (
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
	"gaiasec-nodeagent/pkg/httpsend"
	"gaiasec-nodeagent/pkg/pb"
)

var httpSendService = httpsend.NewHttpSendService()

func (na *NodeAgent) handleHttpSendRequest(message *pb.Base) {
	log.Info("handleHttpSendRequest")

	request := &pb.HttpSendRequestProto{}
	if err := proto.Unmarshal(message.Data, request); err != nil {
		log.Error("Failed to parse HttpSendRequestProto:", err)
		na.sendHttpSendErrorResponse(message, "Failed to parse request: "+err.Error())
		return
	}

	log.Infof("HTTP send request: host=%s, port=%d, secure=%v", request.Host, request.Port, request.Secure)

	response := httpSendService.SendRequest(request)

	err := na.wsClient.SendMessage(response, pb.MessageType_HTTP_SEND_RESPONSE, na.NodeID, message.Source, message.Session)
	if err != nil {
		log.Error("Failed to send HTTP response:", err)
		return
	}

	log.Infof("HTTP send response sent to %s, success=%v", message.Source, response.Success)
}

func (na *NodeAgent) sendHttpSendErrorResponse(message *pb.Base, errMsg string) {
	response := &pb.HttpSendResponseProto{
		Success: false,
		Message: errMsg,
	}

	err := na.wsClient.SendMessage(response, pb.MessageType_HTTP_SEND_RESPONSE, na.NodeID, message.Source, message.Session)
	if err != nil {
		log.Error("Failed to send error response:", err)
	}
}
