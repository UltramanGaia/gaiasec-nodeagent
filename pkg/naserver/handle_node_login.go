package naserver

import (
	log "github.com/sirupsen/logrus"
	"os"
	"runtime"
	"gaiasec-nodeagent/pkg/constant"
	"gaiasec-nodeagent/pkg/pb"
	"gaiasec-nodeagent/pkg/util"
)

func (na *NodeAgent) reportRegister() {
	nodeLogin := &pb.Register{
		Id:           na.NodeID,
		ProjectId:    na.ProjectID,
		ParentId:     "",
		AgentType:    pb.AgentType_NODE_AGENT,
		AgentVersion: "1.0",
		Name:         na.Hostname,
		Hostname:     na.Hostname,
		Ips:          na.IPAddress,
		Os:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		Pid:          int32(os.Getpid()),
	}

	err := na.wsClient.SendMessage(nodeLogin, pb.MessageType_REGISTER, na.NodeID, constant.SERVER_ID, util.GenerateID())
	if err != nil {
		log.Errorf("failed to send node login: %v", err)
		return
	}
}
