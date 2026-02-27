package runtime

// Client abstracts access to the container runtime API for listing containers.
type Client interface {
	ListContainers() ([]Container, error)
	RuntimeType() string
}

// NewClients 创建所有可用的运行时客户端，CRI 优先，Docker 作为回退
func NewClients() []Client {
	var clients []Client
	// CRI 客户端
	criClients := NewCRIClients()
	if len(criClients) > 0 {
		clients = append(clients, criClients...)
	}
	// Docker 回退
	if dockerClient, err := NewDockerClient(); err == nil {
		clients = append(clients, dockerClient)
	}
	return clients
}
