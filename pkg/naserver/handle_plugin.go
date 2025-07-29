package naserver

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
	"sothoth-nodeagent/pkg/pb"
	"sothoth-nodeagent/pkg/plugin"
)

func (na *NodeAgent) handleDeployPluginRequest(message *pb.Base) {
	log.Info("handleDeployPluginRequest")
	request := &pb.DeployPluginRequest{}
	if err := proto.Unmarshal(message.Data, request); err != nil {
		log.Info("parse error:", err)
		return
	}

	err := plugin.DeployPlugin(request)
	if err != nil {
		log.Info("deploy plugin error:", err)
		response := &pb.DeployPluginResponse{
			AgentId:       request.AgentId,
			PluginName:    request.PluginName,
			PluginVersion: request.PluginVersion,
			Pid:           request.Pid,
			Result:        fmt.Sprintf("deploy plugin failed: %v", err),
		}
		err = na.wsClient.SendMessage(response, pb.MessageType_DEPLOY_PLUGIN_RESPONSE, na.NodeID, message.Source, message.Session)
		if err != nil {
			log.Info("send resp error:", err)
			return
		}
		return
	} else {
		log.Info("deploy plugin success")
		response := &pb.DeployPluginResponse{
			AgentId:       request.AgentId,
			PluginName:    request.PluginName,
			PluginVersion: request.PluginVersion,
			Pid:           request.Pid,
			Result:        "deploy plugin success",
		}
		err = na.wsClient.SendMessage(response, pb.MessageType_DEPLOY_PLUGIN_RESPONSE, na.NodeID, message.Source, message.Session)
		if err != nil {
			log.Info("send resp error:", err)
			return
		}
	}

}
