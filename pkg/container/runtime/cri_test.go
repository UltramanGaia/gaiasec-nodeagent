package runtime

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCRIClient_GetContainerList(t *testing.T) {
	t.Skip("Skipping CRI client test - requires containerd/CRI-O runtime environment")
}

func TestCRIClient_getContainerInfo(t *testing.T) {
	t.Skip("Skipping getContainerInfo test - requires CRI runtime environment")
}

func TestCRIClient_convertNetworks(t *testing.T) {
	t.Skip("Skipping convertNetworks test - requires CRI runtime environment")
}

func TestCRIClient_convertAnnotations(t *testing.T) {
	t.Skip("Skipping convertAnnotations test - requires CRI runtime environment")
}

func TestNewCRIClient(t *testing.T) {
	t.Skip("Skipping NewCRIClient test - requires CRI runtime environment")
}

func Test_parseCRIState(t *testing.T) {
	t.Skip("Skipping parseCRIState test - requires CRI runtime environment")
}

func Test_parseK8sMetadata(t *testing.T) {
	t.Skip("Skipping parseK8sMetadata test - requires CRI runtime environment")
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
