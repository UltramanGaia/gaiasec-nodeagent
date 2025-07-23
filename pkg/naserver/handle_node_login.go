package naserver

import (
	log "github.com/sirupsen/logrus"
	"runtime"
	"sothoth-nodeagent/pkg/pb"
)

func (na *NodeAgent) reportNodeLogin() {
	nodeLogin := pb.Register{
		Id:        na.NodeID,
		ProjectId: na.ProjectID,
		Hostname:  na.Hostname,
		Ips:       []string{na.IPAddress},
		Os:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}

	err := na.wsclient.Send(pb.MessageType_REGISTER, &nodeLogin)
	if err != nil {
		log.Errorf("failed to send node login: %v", err)
		return
	}
}
