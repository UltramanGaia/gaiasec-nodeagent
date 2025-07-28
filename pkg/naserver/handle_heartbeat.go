package naserver

import (
	log "github.com/sirupsen/logrus"
	"sothoth-nodeagent/pkg/pb"
	"time"
)

// heartbeatLoop sends periodic heartbeat messages
func (na *NodeAgent) heartbeatLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !na.running {
				return
			}

			heartbeat := pb.Heartbeat{
				Id: na.NodeID,
			}
			err := na.wsClient.Send(pb.MessageType_HEARTBEAT, &heartbeat)
			if err != nil {
				log.Errorf("Heartbeat error: %v", err)
				return
			}
		case <-na.stopChan:
			return
		}
	}
}
