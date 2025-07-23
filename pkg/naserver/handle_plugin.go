package naserver

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
	"sothoth-nodeagent/pkg/pb"
	"sothoth-nodeagent/pkg/plugin"
)

func (na *NodeAgent) handleDeployPluginRequest(message *pb.Base) {
	log.Info("收到插件部署请求")
	request := &pb.DeployPluginRequest{}
	if err := proto.Unmarshal(message.Data, request); err != nil {
		log.Println("解析插件部署请求失败:", err)
		return
	}

	err := plugin.DeployPlugin(request)
	if err != nil {
		log.Println("部署插件失败:", err)
		response := &pb.DeployPluginResponse{
			TaskId:        request.TaskId,
			AgentId:       request.AgentId,
			PluginName:    request.PluginName,
			PluginVersion: request.PluginVersion,
			Pid:           request.Pid,
			Result:        fmt.Sprintf("deploy plugin failed: %v", err),
		}
		na.wsclient.Send(pb.MessageType_DEPLOY_PLUGIN_RESPONSE, response)
		return
	} else {
		log.Println("部署插件成功")
		response := &pb.DeployPluginResponse{
			TaskId:        request.TaskId,
			AgentId:       request.AgentId,
			PluginName:    request.PluginName,
			PluginVersion: request.PluginVersion,
			Pid:           request.Pid,
			Result:        "deploy plugin success",
		}
		na.wsclient.Send(pb.MessageType_DEPLOY_PLUGIN_RESPONSE, response)
	}

}
