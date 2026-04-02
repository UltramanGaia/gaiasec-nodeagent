package runtime

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
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
	log.Infof("[CRI] NewCRIClients start")
	var clients []Client
	for _, cri := range CRISockets {
		log.Infof("[CRI] Trying to connect to %s at %s", cri.name, cri.socket)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		conn, err := grpc.DialContext(ctx, cri.socket, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
		cancel()
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
	log.Infof("[CRI] ListContainers start")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	resp, err := c.client.ListContainers(ctx, &runtimeapi.ListContainersRequest{})
	if err != nil {
		log.Errorf("[CRI] ListContainers failed: %v", err)
		return nil, err
	}
	log.Infof("[CRI] ListContainers got %d containers, fetching status...", len(resp.Containers))

	result := make([]Container, 0, len(resp.Containers))
	for i, ctr := range resp.Containers {
		if i%10 == 0 {
			log.Infof("[CRI] Processing container %d/%d: %s", i+1, len(resp.Containers), ctr.Id)
		}
		status, err := c.client.ContainerStatus(ctx, &runtimeapi.ContainerStatusRequest{ContainerId: ctr.Id, Verbose: true})
		if err != nil {
			log.Warnf("Failed to get status for container %s: %v", ctr.Id, err)
			continue
		}

		pidNs := ""
		pid := 0
		if status.Info != nil {
			if pidStr, ok := status.Info["pid"]; ok {
				pid, _ = strconv.Atoi(pidStr)
				if pid != 0 {
					pidNs, _ = GetPIDNamespace(pid)
				}
			}
		}

		podName, k8sNs := parseK8sMetadata(ctr.Labels)
		state := parseCRIState(status.Status)

		// 从 status.Info 中获取安全相关配置
		privileged := false
		var capAdd, capDrop []string
		if status.Info != nil {
			// 解析 privileged
			if priv, ok := status.Info["privileged"]; ok {
				privileged = priv == "true"
			}
			// 解析 capabilities (从 info 中获取)
			if caps, ok := status.Info["capabilities"]; ok {
				capAdd, capDrop = parseCRICapabilities(caps)
			}
		}

		cont := Container{
			ID:           ctr.Id,
			Name:         ctr.Metadata.Name,
			State:        state,
			ImageID:      stripImageID(ctr.ImageRef),
			ImageName:    status.Status.ImageRef,
			PID:          strconv.Itoa(pid),
			PIDNamespace: pidNs,
			Runtime:      c.runtimeType,
			CreateTime:   status.Status.CreatedAt,
			Labels:       ctr.Labels,
			Annotations:  ctr.Annotations,
			PodName:      podName,
			Namespace:    k8sNs,
			Privileged:   privileged,
			CapAdd:       capAdd,
			CapDrop:      capDrop,
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

// parseCRICapabilities 从 info 字符串中解析 capabilities
func parseCRICapabilities(info string) (capAdd, capDrop []string) {
	// info 可能是 JSON 格式或简单字符串，尝试解析
	// 简化处理：如果包含 add 或 drop 关键词，手动解析
	if strings.Contains(info, "add") || strings.Contains(info, "drop") {
		// 尝试解析 JSON
		var caps map[string][]string
		if err := json.Unmarshal([]byte(info), &caps); err == nil {
			if v, ok := caps["add"]; ok {
				capAdd = v
			}
			if v, ok := caps["drop"]; ok {
				capDrop = v
			}
		}
	}
	return
}
