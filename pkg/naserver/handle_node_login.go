package naserver

import (
	log "github.com/sirupsen/logrus"
	"os"
	"runtime"
	"sothoth-nodeagent/pkg/pb"
)

func (na *NodeAgent) reportRegister() {
	nodeLogin := pb.Register{
		Id:           na.NodeID,
		ProjectId:    na.ProjectID,
		ParentId:     "",
		AgentType:    pb.AgentType_NODE_AGENT,
		AgentVersion: "1.0",
		Hostname:     na.Hostname,
		Ips:          []string{na.IPAddress},
		Os:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		Pid:          int32(os.Getpid()),
	}

	err := na.wsclient.Send(pb.MessageType_REGISTER, &nodeLogin)
	if err != nil {
		log.Errorf("failed to send node login: %v", err)
		return
	}
}
