package naserver

import (
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
	"gaiasec-nodeagent/pkg/pb"
	"gaiasec-nodeagent/pkg/system"
)

func (na *NodeAgent) handleExecuteCommand(message *pb.Base) {
	log.Info("handleExecuteCommand")
	request := &pb.ExecuteCommandRequest{}
	if err := proto.Unmarshal(message.Data, request); err != nil {
		log.Info("parse error:", err)
		return
	}
	result, err := system.ExecuteCommand(request.Command)
	if err != nil {
		log.Info("execute command error:", err)
		return
	}

	response := &pb.ExecuteCommandResponse{
		ExitCode: int32(result.ExitCode),
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
	}

	err = na.wsClient.SendMessage(response, pb.MessageType_EXECUTE_COMMAND_RESPONSE, message.Destination, message.Source, message.Session)
	if err != nil {
		log.Info("send resp error:", err)
		return
	}
}
