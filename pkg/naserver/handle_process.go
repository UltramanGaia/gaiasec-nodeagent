package naserver

import (
	log "github.com/sirupsen/logrus"
	"sothoth-nodeagent/pkg/pb"
	"sothoth-nodeagent/pkg/process"
)

func (na *NodeAgent) handleProcessRequest(message *pb.Base) {
	log.Info("handleProcessRequest")

	processes, err := process.GetProcessList()
	if err != nil {
		log.Info("GetProcesses failed:", err)
	}
	response := &pb.ProcessesResponse{
		Processes: processes,
	}

	err = na.wsClient.SendMessage(response, pb.MessageType_PROCESSES_RESPONSE, message.Destination, message.Source, message.Session)
	if err != nil {
		log.Info("Send error:", err)
		return
	}
}
