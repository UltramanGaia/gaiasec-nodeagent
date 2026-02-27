package runtime

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDockerClient_Integration 测试 Docker 客户端集成
// 这个测试需要实际运行 Docker 环境
func TestDockerClient_Integration(t *testing.T) {
	client, err := NewDockerClient()
	if err != nil {
		t.Skipf("Skipping Docker integration test - cannot connect to Docker daemon: %v", err)
		return
	}

	// 获取容器列表
	containers, err := client.ListContainers()
	if err != nil {
		t.Fatalf("Failed to list containers: %v", err)
	}

	// 验证结果
	t.Logf("Found %d containers", len(containers))

	// 验证运行时类型
	assert.Equal(t, "docker", client.RuntimeType(), "Runtime type should be docker")

	// 如果有容器，验证容器字段
	if len(containers) > 0 {
		t.Run("ContainerFields", func(t *testing.T) {
			for i, c := range containers {
				t.Run(c.Name, func(t *testing.T) {
					assert.NotEmpty(t, c.ID, "Container ID should not be empty")
					assert.NotEmpty(t, c.Name, "Container Name should not be empty")
					assert.NotEmpty(t, c.State, "Container State should not be empty")
					assert.Equal(t, "docker", c.Runtime, "Runtime should be docker")
					assert.NotZero(t, c.CreateTime, "CreateTime should not be zero")

					t.Logf("Container %d: ID=%s, Name=%s, State=%s", i, c.ID[:12], c.Name, c.State)

					// 验证可选字段
					if c.ImageID != "" {
						t.Logf("  Image ID: %s", c.ImageID[:12])
					}
					if c.ImageName != "" {
						t.Logf("  Image Name: %s", c.ImageName)
					}
					if c.PID != "0" {
						t.Logf("  PID: %s", c.PID)
					}
					if len(c.Ports) > 0 {
						t.Logf("  Ports: %d mapping(s)", len(c.Ports))
					}
					if len(c.Mounts) > 0 {
						t.Logf("  Mounts: %d mount(s)", len(c.Mounts))
					}
					if len(c.Networks) > 0 {
						t.Logf("  Networks: %d network(s)", len(c.Networks))
					}
					if len(c.Labels) > 0 {
						t.Logf("  Labels: %d label(s)", len(c.Labels))
					}
				})
			}
		})
	}

	t.Log("Integration test passed")
}
