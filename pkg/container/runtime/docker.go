package runtime

import (
	"context"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
)

type DockerClient struct {
	client *client.Client
}

// 内部运行时容器模型，避免与外部包产生循环引用
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

// NewDockerClient 创建 Docker 客户端
func NewDockerClient() (*DockerClient, error) {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, err
	}

	// 验证连接
	if _, err := cli.Ping(context.Background()); err != nil {
		return nil, err
	}

	return &DockerClient{client: cli}, nil
}

func (d *DockerClient) ListContainers() ([]Container, error) {
	ctx := context.Background()
	ctrs, err := d.client.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return nil, err
	}

	result := make([]Container, 0, len(ctrs))
	for _, ctr := range ctrs {
		inspect, err := d.client.ContainerInspect(ctx, ctr.ID)
		if err != nil {
			log.WithError(err).Warnf("Failed to inspect container %s", ctr.ID)
			continue
		}

		pid := 0
		if inspect.State.Pid != 0 {
			pid = inspect.State.Pid
		}
		pidNs, _ := GetPIDNamespace(pid)

		ports := parseDockerPorts(inspect.NetworkSettings.Ports)
		mounts := parseDockerMounts(inspect.Mounts)
		networks := parseDockerNetworks(inspect.NetworkSettings.Networks)

		cont := Container{
			ID:           ctr.ID,
			Name:         strings.TrimPrefix(ctr.Names[0], "/"),
			State:        ctr.State,
			ImageID:      ctr.ImageID,
			ImageName:    inspect.Config.Image,
			PID:          strconv.Itoa(inspect.State.Pid),
			PIDNamespace: pidNs,
			Runtime:      "docker",
			CreateTime:   ctr.Created.Unix(),
			Ports:        ports,
			Mounts:       mounts,
			Networks:     networks,
			Labels:       inspect.Config.Labels,
		}
		result = append(result, cont)
	}

	return result, nil
}

func (d *DockerClient) RuntimeType() string {
	return "docker"
}

func parseDockerPorts(ports types.PortMap) []PortMapping {
	result := []PortMapping{}
	for containerPort, bindings := range ports {
		for _, binding := range bindings {
			result = append(result, PortMapping{
				ContainerPort: int32(containerPort.Int()),
				Protocol:      containerPort.Proto(),
				HostIP:        binding.HostIP,
				HostPort:      int32(binding.HostPort),
			})
		}
	}
	return result
}

func parseDockerMounts(mounts []types.MountPoint) []MountPoint {
	result := make([]MountPoint, 0, len(mounts))
	for _, mount := range mounts {
		result = append(result, MountPoint{
			Source:      mount.Source,
			Destination: mount.Destination,
			Type:        mount.Type,
			Driver:      mount.Driver,
		})
	}
	return result
}

func parseDockerNetworks(networks map[string]types.NetworkResource) []ContainerNetwork {
	result := make([]ContainerNetwork, 0, len(networks))
	for name, network := range networks {
		result = append(result, ContainerNetwork{
			NetworkID:   network.NetworkID,
			NetworkName: name,
			IPAddress:   network.IPAddress,
			Gateway:     network.Gateway,
			Aliases:     network.Aliases,
		})
	}
	return result
}
