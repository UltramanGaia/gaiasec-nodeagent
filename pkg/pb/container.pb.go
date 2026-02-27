package pb

// Minimal protobuf-like definitions for container information used by the NodeAgent.
// This is a hand-written stand-in for generated pb.go files in this environment.

type Container struct {
	Id           string              `json:"id"`
	Name         string              `json:"name"`
	State        string              `json:"state"`
	ImageId      string              `json:"image_id"`
	ImageName    string              `json:"image_name"`
	Pid          string              `json:"pid"`
	PidNamespace string              `json:"pid_namespace"`
	Runtime      string              `json:"runtime"`
	CreateTime   int64               `json:"create_time"`
	Ports        []*PortMapping      `json:"ports"`
	Mounts       []*MountPoint       `json:"mounts"`
	Networks     []*ContainerNetwork `json:"networks"`
	Labels       map[string]string   `json:"labels"`
	Annotations  map[string]string   `json:"annotations"`
	PodName      string              `json:"pod_name"`
	Namespace    string              `json:"namespace"`
}

type PortMapping struct {
	ContainerPort int32  `json:"container_port"`
	Protocol      string `json:"protocol"`
	HostIp        string `json:"host_ip"`
	HostPort      int32  `json:"host_port"`
}

type MountPoint struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Type        string `json:"type"`
	Driver      string `json:"driver"`
}

type ContainerNetwork struct {
	NetworkId   string   `json:"network_id"`
	NetworkName string   `json:"network_name"`
	IpAddress   string   `json:"ip_address"`
	Gateway     string   `json:"gateway"`
	Aliases     []string `json:"aliases"`
}

// ContainerResponse is a protobuf-like wrapper for container list responses
type ContainerResponse struct {
	Containers []*Container `json:"containers"`
}
