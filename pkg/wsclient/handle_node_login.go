package wsclient

import (
	log "github.com/sirupsen/logrus"
	"runtime"
	"sothoth-nodeagent/pkg/pb"
	"sothoth-nodeagent/pkg/system"
)

func (c *Client) reportNodeLogin() {
	hostname, err := system.GetHostname()
	if err != nil {
		return
	}

	ipAddress, err := system.GetLocalIP()
	if err != nil {
		return
	}

	nodeLogin := pb.NodeLogin{
		ProjectId: c.cfg.ProjectID,
		NodeId:    c.cfg.NodeID,
		Hostname:  hostname,
		Ip:        ipAddress,
		Os:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}

	err = c.Send(pb.MessageType_NODE_LOGIN, &nodeLogin)
	if err != nil {
		log.Errorf("failed to send node login: %v", err)
		return
	}
}
