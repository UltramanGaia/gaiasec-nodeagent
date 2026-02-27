package runtime

import (
	"testing"

	"gaiasec-nodeagent/pkg/container"

	"github.com/stretchr/testify/assert"
)

func TestDockerClient_GetContainerList(t *testing.T) {
	client, err := NewDockerClient()
	if err != nil {
		t.Skipf("Skipping Docker client test - cannot connect to Docker daemon: %v", err)
		return
	}

	containers, err := client.ListContainers()
	if err != nil {
		t.Fatalf("Failed to get container list: %v", err)
	}

	assert.NotEmpty(t, containers, "Should find at least one container")

	for _, c := range containers {
		assert.NotEmpty(t, c.ID, "Container ID should not be empty")
		assert.NotEmpty(t, c.Name, "Container Name should not be empty")
		assert.NotEmpty(t, c.State, "Container State should not be empty")
		assert.Equal(t, "docker", c.Runtime, "Runtime should be docker")
		assert.NotZero(t, c.CreateTime, "CreateTime should not be zero")
	}

	t.Logf("Successfully collected %d containers", len(containers))
}

func TestDockerClient_getContainerInfo(t *testing.T) {
	t.Skip("Skipping getContainerInfo test - requires Docker runtime environment")

	// Example implementation:
	/*
		client := &DockerClient{client: mockDockerClient()}
		container := &types.Container{
			ID:    "test-id",
			Names: []string{"/test-container"},
			State: "running",
		}

		info, err := client.getContainerInfo(container)
		assert.NoError(t, err)
		assert.Equal(t, "test-id", info.ID)
		assert.Equal(t, "test-container", info.Name)
	*/
}

func TestDockerClient_convertNetworks(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]dockerNetwork
		expected func(t *testing.T, networks []*container.ContainerNetwork)
	}{
		{
			name: "single network",
			input: map[string]dockerNetwork{
				"bridge": {
					IPAMConfig: dockerIPAMConfig{
						IPv4Address: "172.17.0.2",
					},
					MacAddress: "02:42:ac:11:00:02",
				},
			},
			expected: func(t *testing.T, networks []*container.ContainerNetwork) {
				assert.Len(t, networks, 1)
				assert.Equal(t, "bridge", networks[0].Name)
				assert.Equal(t, "172.17.0.2", networks[0].IPAddress)
				assert.Equal(t, "02:42:ac:11:00:02", networks[0].MACAddress)
			},
		},
		{
			name: "multiple networks",
			input: map[string]dockerNetwork{
				"bridge": {
					IPAMConfig: dockerIPAMConfig{
						IPv4Address: "172.17.0.2",
					},
					MacAddress: "02:42:ac:11:00:02",
				},
				"host": {
					MacAddress: "",
				},
			},
			expected: func(t *testing.T, networks []*container.ContainerNetwork) {
				assert.Len(t, networks, 2)
				// Check both networks exist
				found := make(map[string]bool)
				for _, net := range networks {
					found[net.Name] = true
				}
				assert.True(t, found["bridge"])
				assert.True(t, found["host"])
			},
		},
		{
			name:  "empty networks",
			input: map[string]dockerNetwork{},
			expected: func(t *testing.T, networks []*container.ContainerNetwork) {
				assert.Empty(t, networks)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &DockerClient{}
			networks := client.convertNetworks(tt.input)
			tt.expected(t, networks)
		})
	}
}

func TestDockerClient_convertPorts(t *testing.T) {
	tests := []struct {
		name     string
		input    []dockerPort
		expected func(t *testing.T, ports []*container.PortMapping)
	}{
		{
			name: "single port mapping",
			input: []dockerPort{
				{
					PublicPort:  8080,
					PrivatePort: 80,
					Type:        "tcp",
					IP:          "0.0.0.0",
				},
			},
			expected: func(t *testing.T, ports []*container.PortMapping) {
				assert.Len(t, ports, 1)
				assert.Equal(t, int32(8080), ports[0].HostPort)
				assert.Equal(t, int32(80), ports[0].ContainerPort)
				assert.Equal(t, "tcp", ports[0].Protocol)
				assert.Equal(t, "0.0.0.0", ports[0].HostIP)
			},
		},
		{
			name: "multiple port mappings",
			input: []dockerPort{
				{
					PublicPort:  8080,
					PrivatePort: 80,
					80,
					Type: "tcp",
					IP:   "0.0.0.0",
				},
				{
					PublicPort:  443,
					PrivatePort: 443,
					Type:        "tcp",
					IP:          "0.0.0.0",
				},
			},
			expected: func(t *testing.T, ports []*container.PortMapping) {
				assert.Len(t, ports, 2)
			},
		},
		{
			name:  "no ports",
			input: []dockerPort{},
			expected: func(t *testing.T, ports []*container.PortMapping) {
				assert.Empty(t, ports)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &DockerClient{}
			ports := client.convertPorts(tt.input)
			tt.expected(t, ports)
		})
	}
}

func TestDockerClient_convertMounts(t *testing.T) {
	tests := []struct {
		name     string
		input    []dockerMount
		expected func(t *testing.T, mounts []*container.MountPoint)
	}{
		{
			name: "volume mount",
			input: []dockerMount{
				{
					Type:        "volume",
					Source:      "/var/lib/docker/volumes/data/_data",
					Destination: "/data",
					RW:          true,
				},
			},
			expected: func(t *testing.T, mounts []*container.MountPoint) {
				assert.Len(t, mounts, 1)
				assert.Equal(t, "volume", mounts[0].Type)
				assert.Equal(t, "/var/lib/docker/volumes/data/_data", mounts[0].Source)
				assert.Equal(t, "/data", mounts[0].Destination)
				assert.True(t, mounts[0].RW)
			},
		},
		{
			name: "bind mount",
			input: []dockerMount{
				{
					Type:        "bind",
					Source:      "/host/path",
					Destination: "/container/path",
					RW:          true,
				},
			},
			expected: func(t *testing.T, mounts []*container.MountPoint) {
				assert.Len(t, mounts, 1)
				assert.Equal(t, "bind", mounts[0].Type)
				assert.Equal(t, "/host/path", mounts[0].Source)
				assert.Equal(t, "/container/path", mounts[0].Destination)
			},
		},
		{
			name: "read-only mount",
			input: []dockerMount{
				{
					Type:        "bind",
					Source:      "/readonly/path",
					Destination: "/ro/path",
					RW:          false,
				},
			},
			expected: func(t *testing.T, mounts []*container.MountPoint) {
				assert.Len(t, mounts, 1)
				assert.False(t, mounts[0].RW)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &DockerClient{}
			mounts := client.convertMounts(tt.input)
			tt.expected(t, mounts)
		})
	}
}
