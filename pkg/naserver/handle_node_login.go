package naserver

import (
	log "github.com/sirupsen/logrus"
	"runtime"
	"sothoth-nodeagent/pkg/pb"
)

func (na *NodeAgent) reportNodeLogin() {
	nodeLogin := pb.NodeLogin{
		ProjectId: na.ProjectID,
		NodeId:    na.NodeID,
		Hostname:  na.Hostname,
		Ip:        na.IPAddress,
		Os:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}

	err := na.wsclient.Send(pb.MessageType_NODE_LOGIN, &nodeLogin)
	if err != nil {
		log.Errorf("failed to send node login: %v", err)
		return
	}
}
