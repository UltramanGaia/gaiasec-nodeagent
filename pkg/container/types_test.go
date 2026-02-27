package container

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToProtobufContainer(t *testing.T) {
	tests := []struct {
		name     string
		input    *Container
		expected func(t *testing.T, pbContainer *protobufContainer)
	}{
		{
			name: "basic container",
			input: &Container{
				ID:          "container-123",
				Name:        "test-container",
				Status:      "running",
				Image:       "nginx:latest",
				ImageID:     "sha256:abc123",
				Runtime:     "docker",
				RuntimePath: "/var/run/docker.sock",
			},
			expected: func(t *testing.T, pb *protobufContainer) {
				assert.Equal(t, "container-123", pb.Id)
				assert.Equal(t, "test-container", pb.Name)
				assert.Equal(t, "running", pb.Status)
				assert.Equal(t, "nginx:latest", pb.Image)
				assert.Equal(t, "sha256:abc123", pb.ImageId)
				assert.Equal(t, "docker", pb.Runtime)
				assert.Equal(t, "/var/run/docker.sock", pb.RuntimePath)
			},
		},
		{
			name: "container with network config",
			input: &Container{
				ID:     "container-456",
				Name:   "web-server",
				Status: "running",
				Image:  "nginx:alpine",
				Networks: []*ContainerNetwork{
					{
						Name:       "bridge",
						IPAddress:  "172.17.0.2",
						MacAddress: "02:42:ac:11:00:02",
					},
				},
				Ports: []*PortMapping{
					{
						HostPort:      8080,
						ContainerPort: 80,
						Protocol:      "tcp",
						HostIP:        "0.0.0.0",
					},
				},
			},
			expected: func(t *testing.T, pb *protobufContainer) {
				assert.Len(t, pb.Networks, 1)
				assert.Equal(t, "bridge", pb.Networks[0].Name)
				assert.Equal(t, "172.17.0.2", pb.Networks[0].IpAddress)
				assert.Equal(t, "02:42:ac:11:00:02", pb.Networks[0].MacAddress)

				assert.Len(t, pb.Ports, 1)
				assert.Equal(t, int32(8080), pb.Ports[0].HostPort)
				assert.Equal(t, int32(80), pb.Ports[0].ContainerPort)
				assert.Equal(t, "tcp", pb.Ports[0].Protocol)
				assert.Equal(t, "0.0.0.0", pb.Ports[0].HostIp)
			},
		},
		{
			name: "container with mounts",
			input: &Container{
				ID:     "container-789",
				Name:   "data-volume",
				Status: "running",
				Image:  "redis:latest",
				Mounts: []*MountPoint{
					{
						Type:        "volume",
						Source:      "/var/lib/docker/volumes/data/_data",
						Destination: "/data",
						RW:          true,
					},
				},
			},
			expected: func(t *testing.T, pb *protobufContainer) {
				assert.Len(t, pb.Mounts, 1)
				assert.Equal(t, "volume", pb.Mounts[0].Type)
				assert.Equal(t, "/var/lib/docker/volumes/data/_data", pb.Mounts[0].Source)
				assert.Equal(t, "/data", pb.Mounts[0].Destination)
				assert.True(t, pb.Mounts[0].Rw)
			},
		},
		{
			name: "container with labels",
			input: &Container{
				ID:     "container-labels",
				Name:   "k8s-pod",
				Status: "running",
				Image:  "app:v1.2.3",
				Labels: map[string]string{
					"io.kubernetes.pod.name":      "my-pod",
					"io.kubernetes.pod.namespace": "default",
					"app.kubernetes.io/name":      "my-app",
				},
				Annotations: map[string]string{
					"kubectl.kubernetes.io/last-applied-configuration": "config",
				},
			},
			expected: func(t *testing.T, pb *protobufContainer) {
				assert.Len(t, pb.Labels, 3)
				assert.Equal(t, "my-pod", pb.Labels["io.kubernetes.pod.name"])
				assert.Equal(t, "default", pb.Labels["io.kubernetes.pod.namespace"])
				assert.Equal(t, "my-app", pb.Labels["app.kubernetes.io/name"])

				assert.Len(t, pb.Annotations, 1)
				assert.Equal(t, "config", pb.Annotations["kubectl.kubernetes.io/last-applied-configuration"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toProtobufContainer(tt.input)
			tt.expected(t, result)
		})
	}
}
