package container

import (
	"gaiasec-nodeagent/pkg/container/runtime"
	"gaiasec-nodeagent/pkg/pb"
	log "github.com/sirupsen/logrus"
)

// Internal runtime container representations
type PortMapping struct {
	ContainerPort int32
	Protocol      string
	HostIP        string
	HostPort      int32
}

type MountPoint struct {
	Source      string
	Destination string
	Type        string
	Driver      string
}

type ContainerNetwork struct {
	NetworkID   string
	NetworkName string
	IPAddress   string
	Gateway     string
	Aliases     []string
}

// Container is an internal representation used to collect data from runtime clients
type Container struct {
	ID           string
	Name         string
	State        string
	ImageID      string
	ImageName    string
	PID          string
	PIDNamespace string
	Runtime      string
	CreateTime   int64
	Ports        []PortMapping
	Mounts       []MountPoint
	Networks     []ContainerNetwork
	Labels       map[string]string
	Annotations  map[string]string
	PodName      string
	Namespace    string
}

// GetContainerList 获取所有运行时的容器列表
func GetContainerList() ([]*pb.Container, error) {
	// 获取所有运行时客户端
	clients := runtime.NewClients()
	if len(clients) == 0 {
		log.Warn("No container runtime clients available")
		return []*pb.Container{}, nil
	}

	log.Infof("Found %d container runtime clients", len(clients))

	// 从所有客户端收集容器信息
	var allContainers []*Container
	for _, client := range clients {
		containers, err := client.ListContainers()
		if err != nil {
			log.Warnf("Failed to list containers from %s: %v", client.RuntimeType(), err)
			continue
		}
		log.Infof("Collected %d containers from %s", len(containers), client.RuntimeType())
		allContainers = append(allContainers, containers...)
	}

	// 转换为 protobuf 结构
	return toProtobuf(allContainers), nil
}

// 将内部 Container 转换为 protobuf Container 列表
func toProtobuf(containers []*Container) []*pb.Container {
	result := make([]*pb.Container, 0, len(containers))
	for _, c := range containers {
		pbContainer := &pb.Container{
			Id:           c.ID,
			Name:         c.Name,
			State:        c.State,
			ImageId:      c.ImageID,
			ImageName:    c.ImageName,
			Pid:          c.PID,
			PidNamespace: c.PIDNamespace,
			Runtime:      c.Runtime,
			CreateTime:   c.CreateTime,
			Ports:        toProtobufPorts(c.Ports),
			Mounts:       toProtobufMounts(c.Mounts),
			Networks:     toProtobufNetworks(c.Networks),
			Labels:       c.Labels,
			Annotations:  c.Annotations,
			PodName:      c.PodName,
			Namespace:    c.Namespace,
		}
		result = append(result, pbContainer)
	}
	return result
}

func toProtobufPorts(ports []PortMapping) []*pb.PortMapping {
	result := make([]*pb.PortMapping, 0, len(ports))
	for _, p := range ports {
		result = append(result, &pb.PortMapping{
			ContainerPort: p.ContainerPort,
			Protocol:      p.Protocol,
			HostIp:        p.HostIP,
			HostPort:      p.HostPort,
		})
	}
	return result
}

func toProtobufMounts(mounts []MountPoint) []*pb.MountPoint {
	result := make([]*pb.MountPoint, 0, len(mounts))
	for _, m := range mounts {
		result = append(result, &pb.MountPoint{
			Source:      m.Source,
			Destination: m.Destination,
			Type:        m.Type,
			Driver:      m.Driver,
		})
	}
	return result
}

func toProtobufNetworks(networks []ContainerNetwork) []*pb.ContainerNetwork {
	result := make([]*pb.ContainerNetwork, 0, len(networks))
	for _, n := range networks {
		result = append(result, &pb.ContainerNetwork{
			NetworkId:   n.NetworkID,
			NetworkName: n.NetworkName,
			IpAddress:   n.IPAddress,
			Gateway:     n.Gateway,
			Aliases:     n.Aliases,
		})
	}
	return result
}
