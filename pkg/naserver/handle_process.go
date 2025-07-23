package naserver

import (
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
	"sothoth-nodeagent/pkg/pb"
	"sothoth-nodeagent/pkg/process"
)

func (na *NodeAgent) handleProcessRequest(message *pb.Base) {
	log.Info("收到进程列表请求")
	request := &pb.ProcessesRequest{}
	if err := proto.Unmarshal(message.Data, request); err != nil {
		log.Println("解析进程列表请求失败:", err)
		return
	}

	processes, err := process.GetProcessList()
	if err != nil {
		log.Info("GetProcesses failed:", err)
	}
	response := &pb.ProcessesResponse{
		TaskId:    request.TaskId,
		Processes: processes,
	}

	data, err := proto.Marshal(response)
	if err != nil {
		log.Println("序列化进程列表响应失败:", err)
		return
	}

	msg := pb.Base{
		Type:    pb.MessageType_PROCESSES_RESPONSE,
		Session: message.Session,
		Data:    data,
	}

	bytes, err := proto.Marshal(&msg)
	if err != nil {
		log.Println("序列化进程列表响应失败:", err)
		return
	}
	err = na.wsclient.SendMessage(bytes)
	if err != nil {
		log.Println("发送进程列表响应失败:", err)
		return
	}
}
