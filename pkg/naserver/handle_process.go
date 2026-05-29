package naserver

import (
	"gaiasec-nodeagent/pkg/pb"
	"gaiasec-nodeagent/pkg/process"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
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

func (na *NodeAgent) handleProcessMetadataRequest(message *pb.Base) {
	request := &pb.ProcessMetadataRequest{}
	if err := proto.Unmarshal(message.GetData(), request); err != nil {
		log.Info("decode process metadata request failed:", err)
		return
	}

	response, err := process.GetProcessMetadata(request.GetPid())
	if err != nil {
		log.Info("GetProcessMetadata failed:", err)
		response = &pb.ProcessMetadataResponse{
			Pid: request.GetPid(),
		}
	}

	if err := na.wsClient.SendMessage(response, pb.MessageType_PROCESS_METADATA_RESPONSE, message.Destination, message.Source, message.Session); err != nil {
		log.Info("Send error:", err)
	}
}
