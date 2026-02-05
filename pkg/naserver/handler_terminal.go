package naserver

import (
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
	"gaiasec-nodeagent/pkg/pb"
	"gaiasec-nodeagent/pkg/terminal"
)

// PTY相关处理方法
func (na *NodeAgent) handlePtyCreate(message *pb.Base) {
	msg := &pb.TerminalCreateRequest{}
	err := proto.Unmarshal(message.Data, msg)
	if err != nil {
		log.Info("unmarshal error:", err)
		return
	}

	terminalServer := terminal.NewTerminal(int(msg.Cols), int(msg.Rows), msg.Command, msg.Session)
	go terminalServer.Start()

	response := &pb.TerminalCreateResponse{
		Result: "success",
	}
	err = na.wsClient.SendMessage(response, pb.MessageType_TERMINAL_CREATE_RESPONSE, message.Destination, message.Source, message.Session)
	if err != nil {
		log.Info("send resp error:", err)
		return
	}
}
