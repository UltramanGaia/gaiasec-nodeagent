package runtime

import (
	"testing"
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
	t.Skip("Skipping NewCRIClient test - - requires CRI runtime environment")
}
