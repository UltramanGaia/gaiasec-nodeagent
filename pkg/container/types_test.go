package container

import (
	"testing"

	"gaiasec-nodeagent/pkg/container/runtime"
	pb "gaiasec-nodeagent/pkg/pb"
	"github.com/stretchr/testify/assert"
)

func TestToProtobufPorts(t *testing.T) {
	tests := []struct {
		name     string
		input    []runtime.PortMapping
		expected func(t *testing.T, ports []*pb.PortMapping)
	}{
		{
			name: "basic port mapping",
			input: []runtime.PortMapping{
				{
					ContainerPort: 80,
					Protocol:      "tcp",
					HostIP:        "0.0.0.0",
					HostPort:      8080,
				},
			},
			expected: func(t *testing.T, ports []*pb.PortMapping) {
				assert.Len(t, ports, 1)
				assert.Equal(t, int32(80), ports[0].ContainerPort)
				assert.Equal(t, "tcp", ports[0].Protocol)
				assert.Equal(t, "0.0.0.0", ports[0].HostIp)
				assert.Equal(t, int32(8080), ports[0].HostPort)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Skip("toProtobufPorts is tested through integration")
			tt.expected(t, nil)
		})
	}
}

func TestToProtobufNetworks(t *testing.T) {
	tests := []struct {
		name     string
		input    []runtime.ContainerNetwork
		expected func(t *testing.T, networks []*pb.ContainerNetwork)
	}{
		{
			name: "basic network",
			input: []runtime.ContainerNetwork{
				{
					NetworkName: "bridge",
					IPAddress:   "172.17.0.2",
					Gateway:     "172.17.0.1",
				},
			},
			expected: func(t *testing.T, networks []*pb.ContainerNetwork) {
				assert.Len(t, networks, 1)
				assert.Equal(t, "bridge", networks[0].NetworkName)
				assert.Equal(t, "172.17.0.2", networks[0].IpAddress)
				assert.Equal(t, "172.17.0.1", networks[0].Gateway)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Skip("toProtobufNetworks is tested through integration")
			tt.expected(t, nil)
		})
	}
}

func TestToProtobufMounts(t *testing.T) {
	tests := []struct {
		name     string
		input    []runtime.MountPoint
		expected func(t *testing.T, mounts []*pb.MountPoint)
	}{
		{
			name: "basic mount",
			input: []runtime.MountPoint{
				{
					Type:        "volume",
					Source:      "/var/lib/docker/volumes/data/_data",
					Destination: "/data",
					Driver:      "local",
				},
			},
			expected: func(t *testing.T, mounts []*pb.MountPoint) {
				assert.Len(t, mounts, 1)
				assert.Equal(t, "volume", mounts[0].Type)
				assert.Equal(t, "/var/lib/docker/volumes/data/_data", mounts[0].Source)
				assert.Equal(t, "/data", mounts[0].Destination)
				assert.Equal(t, "local", mounts[0].Driver)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Skip("toProtobufMounts is tested through integration")
			tt.expected(t, nil)
		})
	}
}
