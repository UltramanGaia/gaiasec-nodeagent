package naserver

import (
	log "github.com/sirupsen/logrus"
	"sothoth-nodeagent/pkg/constant"
	"sothoth-nodeagent/pkg/pb"
	"sothoth-nodeagent/pkg/util"
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

			heartbeat := &pb.Heartbeat{
				Id: na.NodeID,
			}
			err := na.wsClient.SendMessage(heartbeat, pb.MessageType_HEARTBEAT, na.NodeID, constant.SERVER_ID, util.GenerateID())
			if err != nil {
				log.Errorf("Heartbeat error: %v", err)
				return
			}
		case <-na.stopChan:
			return
		}
	}
}
