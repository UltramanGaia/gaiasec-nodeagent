package naserver

import (
	log "github.com/sirupsen/logrus"
	"sothoth-nodeagent/pkg/network"
	"sothoth-nodeagent/pkg/pb"
)

// handleNetworkRequest handles the network connection information request
func (na *NodeAgent) handleNetworkRequest(message *pb.Base) {
	log.Info("handleNetworkRequest")

	// Get network connections
	connections, err := network.GetNetworkConnections()
	if err != nil {
		log.Error("GetNetworkConnections failed:", err)
		// Create an empty response if we can't get connections
		connections = []*pb.NetworkConnection{}
	}

	// Create response
	response := &pb.NetworkResponse{
		Connections: connections,
	}

	// Send response
	err = na.wsClient.SendMessage(response, pb.MessageType_NETWORK_RESPONSE, message.Destination, message.Source, message.Session)
	if err != nil {
		log.Error("Send network response error:", err)
		return
	}
}