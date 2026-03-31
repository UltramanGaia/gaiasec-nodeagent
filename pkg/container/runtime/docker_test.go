package runtime

import (
	"testing"

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

	t.Logf("Successfully collected %d containers", len(containers))

	for _, c := range containers {
		assert.NotEmpty(t, c.ID, "Container ID should not be empty")
		assert.NotEmpty(t, c.Name, "Container Name should not be empty")
		assert.NotEmpty(t, c.State, "Container State should not be empty")
		assert.Equal(t, "docker", c.Runtime, "Runtime should be docker")
		assert.NotZero(t, c.CreateTime, "CreateTime should not be zero")
	}
}

func TestDockerClient_getContainerInfo(t *testing.T) {
	t.Skip("Skipping getContainerInfo test - requires Docker runtime environment")
}

func TestDockerClient_convertNetworks(t *testing.T) {
	t.Skip("Skipping convertNetworks test - requires Docker runtime environment")
}

func TestDockerClient_convertPorts(t *testing.T) {
	t.Skip("Skipping convertPorts test - requires Docker runtime environment")
}

func TestDockerClient_convertMounts(t *testing.T) {
	t.Skip("Skipping convertMounts test - requires Docker runtime environment")
}

func Test_parseDockerState(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "running state",
			input:    "running",
			expected: "running",
		},
		{
			name:     "exited state",
			input:    "exited",
			expected: "exited",
		},
		{
			name:     "dead state",
			input:    "dead",
			expected: "dead",
		},
		{
			name:     "empty state",
			input:    "",
			expected: "unknown",
		},
		{
			name:     "unknown state",
			input:    "weird-state",
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseDockerState(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func Test_stripImageID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "sha256 prefix",
			input:    "sha256:abc123def456",
			expected: "abc123def456",
		},
		{
			name:     "no prefix",
			input:    "abc123def456",
			expected: "abc123def456",
		},
		{
			name:     "empty",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripImageID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewDockerClient(t *testing.T) {
	client, err := NewDockerClient()
	if err != nil {
		t.Skipf("Skipping NewDockerClient test - cannot connect to Docker daemon: %v", err)
		return
	}
	assert.NotNil(t, client)
}
