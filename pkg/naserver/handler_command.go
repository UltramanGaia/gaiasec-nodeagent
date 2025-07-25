package naserver

import (
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
	"sothoth-nodeagent/pkg/pb"
	"sothoth-nodeagent/pkg/system"
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

	data, err := proto.Marshal(response)
	if err != nil {
		log.Info("Marshal error:", err)
		return
	}

	msg := pb.Base{
		Type:    pb.MessageType_EXECUTE_COMMAND_RESPONSE,
		Session: message.Session,
		Data:    data,
	}

	bytes, err := proto.Marshal(&msg)
	if err != nil {
		log.Info("Marshal error:", err)
		return
	}
	err = na.wsclient.SendMessage(bytes)
	if err != nil {
		log.Info("send resp error:", err)
		return
	}
}
