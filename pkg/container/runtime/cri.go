package runtime

import (
	"context"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
	"strconv"
	"strings"
)

// CRIClient 实现了运行时客户端接口（CRI 通过 CRI-O/containerd 的 CRI API）
type CRIClient struct {
	client      runtimeapi.RuntimeServiceClient
	runtimeType string
}

// CRISockets 预定义的 CRI 套接字及其运行时类型
var CRISockets = []struct {
	name    string
	socket  string
	runtime string
}{
	{"containerd", "unix:///run/containerd/containerd.sock", "containerd"},
	{"crio", "unix:///run/crio/crio.sock", "crio"},
	{"cri-dockerd", "unix:///var/run/cri-dockerd.sock", "cri-dockerd"},
}

// NewCRIClients 尝试连接可用的 CRI 实现，返回可用的客户端切片
func NewCRIClients() []Client {
	var clients []Client
	for _, cri := range CRISockets {
		conn, err := grpc.Dial(cri.socket, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
		if err != nil {
			log.Debugf("Failed to connect to %s: %v", cri.name, err)
			continue
		}
		c := &CRIClient{client: runtimeapi.NewRuntimeServiceClient(conn), runtimeType: cri.runtime}
		clients = append(clients, c)
		log.Infof("Connected to CRI runtime: %s", cri.runtime)
	}
	return clients
}

// ListContainersList 实现 CRI 客户端容器列表获取逻辑
func (c *CRIClient) ListContainers() ([]Container, error) {
	ctx := context.Background()
	resp, err := c.client.ListContainers(ctx, &runtimeapi.ListContainersRequest{})
	if err != nil {
		return nil, err
	}

	result := make([]Container, 0, len(resp.Containers))
	for _, ctr := range resp.Containers {
		status, err := c.client.ContainerStatus(ctx, &runtimeapi.ContainerStatusRequest{ContainerId: ctr.Id, Verbose: true})
		if err != nil {
			log.Warnf("Failed to get status for container %s: %v", ctr.Id, err)
			continue
		}

		pidNs := ""
		if status.Status != nil && status.Status.GetProcessConfig() != nil {
			// 兼容性处理：某些版本字段不会直接暴露 ProcessId
		}
		if status.Status != nil && status.Status.GetPid() != 0 {
			pidNs, _ = GetPIDNamespace(int(status.Status.Pid))
		}

		podName, k8sNs := parseK8sMetadata(status.Labels)
		state := parseCRIState(status.Status)

		cont := Container{
			ID:           ctr.Id,
			Name:         ctr.Metadata.Name,
			State:        state,
			ImageID:      stripImageID(ctr.ImageRef),
			ImageName:    status.Status.GetImageRef(),
			PID:          strconv.Itoa(int(status.Status.GetPid())),
			PIDNamespace: pidNs,
			Runtime:      c.runtimeType,
			CreateTime:   status.CreatedAt,
			Labels:       status.Labels,
			Annotations:  status.Annotations,
			PodName:      podName,
			Namespace:    k8sNs,
		}
		result = append(result, cont)
	}
	return result, nil
}

func (c *CRIClient) RuntimeType() string {
	return c.runtimeType
}

// Helper helpers 一个简化实现的状态映射
func parseCRIState(s *runtimeapi.ContainerStatus) string {
	if s == nil {
		return "unknown"
	}
	switch s.GetState() {
	case runtimeapi.ContainerState_CONTAINER_CREATED:
		return "created"
	case runtimeapi.ContainerState_CONTAINER_RUNNING:
		return "running"
	case runtimeapi.ContainerState_CONTAINER_EXITED:
		return "exited"
	default:
		return "unknown"
	}
}

func parseK8sMetadata(labels map[string]string) (podName, namespace string) {
	podName = labels["io.kubernetes.pod.name"]
	namespace = labels["io.kubernetes.pod.namespace"]
	return podName, namespace
}

func parseImageName(imageRef string) string {
	return imageRef
}

func stripImageID(imageRef string) string {
	if strings.HasPrefix(imageRef, "sha256:") {
		return strings.TrimPrefix(imageRef, "sha256:")
	}
	return imageRef
}
