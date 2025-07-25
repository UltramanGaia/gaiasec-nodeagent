package naserver

import (
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
	"sothoth-nodeagent/pkg/pb"
	"sothoth-nodeagent/pkg/process"
)

func (na *NodeAgent) handleProcessRequest(message *pb.Base) {
	log.Info("handleProcessRequest ")

	processes, err := process.GetProcessList()
	if err != nil {
		log.Info("GetProcesses failed:", err)
	}
	response := &pb.ProcessesResponse{
		Processes: processes,
	}

	data, err := proto.Marshal(response)
	if err != nil {
		log.Info("Marshal error:", err)
		return
	}

	msg := pb.Base{
		Type:    pb.MessageType_PROCESSES_RESPONSE,
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
		log.Info("Send error:", err)
		return
	}
}
