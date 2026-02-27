package container

// Internal container representations used by the runtime implementations.

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
