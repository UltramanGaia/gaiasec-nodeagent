package container

import (
	"gaiasec-nodeagent/pkg/container/runtime"
	pb "gaiasec-nodeagent/pkg/pb"
	log "github.com/sirupsen/logrus"
)

// GetContainerList 获取所有运行时的容器信息，统一输出为 protobuf 的 Container 列表
func GetContainerList() ([]*pb.Container, error) {
	clients := runtime.NewClients()
	if len(clients) == 0 {
		log.Warn("No container runtime clients available")
		return []*pb.Container{}, nil
	}

	log.Infof("Found %d container runtime clients", len(clients))

	var allContainers []*pb.Container
	for _, cli := range clients {
		containers, err := cli.ListContainers()
		if err != nil {
			log.Warnf("Failed to list containers from %s: %v", cli.RuntimeType(), err)
			continue
		}
		log.Infof("Collected %d containers from %s", len(containers), cli.RuntimeType())
		for i := range containers {
			c := containers[i]
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
			allContainers = append(allContainers, pbContainer)
		}
	}
	return allContainers, nil
}

// Helper: 将内部 Ports 映射为 protobuf 类型
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

// Helper: 将内部 MountPoint 映射为 protobuf 类型
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

// Helper: 将内部 Network 映射为 protobuf 类型
func toProtobufNetworks(nets []ContainerNetwork) []*pb.ContainerNetwork {
	result := make([]*pb.ContainerNetwork, 0, len(nets))
	for _, n := range nets {
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
