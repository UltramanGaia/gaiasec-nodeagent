package escape

import (
	"gaiasec-nodeagent/pkg/container"
	"gaiasec-nodeagent/pkg/pb"
	"strings"

	log "github.com/sirupsen/logrus"
)

// AnalyzeContainerEscape 分析容器逃逸风险
func AnalyzeContainerEscape() ([]*pb.ContainerEscapeInfo, error) {
	log.Info("AnalyzeContainerEscape: calling GetContainerList")
	// 获取容器列表
	containers, err := container.GetContainerList()
	log.Infof("AnalyzeContainerEscape: GetContainerList returned %d containers, err=%v", len(containers), err)
	if err != nil {
		return nil, err
	}

	var results []*pb.ContainerEscapeInfo
	for _, c := range containers {
		info := analyzeSingleContainer(c)
		results = append(results, info)
	}

	return results, nil
}

// analyzeSingleContainer 分析单个容器的逃逸风险
func analyzeSingleContainer(c *pb.Container) *pb.ContainerEscapeInfo {
	info := &pb.ContainerEscapeInfo{
		ContainerId:   c.Id,
		ContainerName: c.Name,
		HostPid:       false,
		HostNetwork:   false,
		HostIpc:       false,
		HostUts:       false,
		Privileged:    false,
		User:          getContainerUser(c),
		NetworkMode:   getNetworkMode(c),
		Runtime:       c.Runtime,
		Image:         c.ImageName,
	}

	// 检查 label 中是否配置了共享宿主机命名空间
	if labels := c.Labels; labels != nil {
		info.HostPid = getBoolLabel(labels, "hostPID")
		info.HostNetwork = getBoolLabel(labels, "hostNetwork")
		info.HostIpc = getBoolLabel(labels, "hostIPC")
		info.HostUts = getBoolLabel(labels, "hostUTS")
	}
	// 优先使用 Container.Privileged 字段（从 Docker HostConfig 获取）
	if c.Privileged {
		info.Privileged = true
	}

	// 分析挂载信息
	info.Mounts = analyzeMounts(c.Mounts)

	// 分析 capabilities (传入 capAdd 和 capDrop)
	info.Capabilities = analyzeCapabilities(c.CapAdd, c.CapDrop)

	// 初步风险评估
	info.HasHighRisk, info.RiskSummary = evaluateContainerRisk(info)

	return info
}

// getBoolLabel 从 labels 中获取布尔值
func getBoolLabel(labels map[string]string, key string) bool {
	if val, ok := labels[key]; ok {
		return strings.ToLower(val) == "true"
	}
	return false
}

// getContainerUser 获取容器运行用户
func getContainerUser(c *pb.Container) string {
	// 从 annotation 或其他信息获取
	if c.Annotations != nil {
		if user, ok := c.Annotations["run.containers.kubernetes.io/container.seccomp.securityAlpha.kubernetes.io/acceleration"]; ok {
			return user
		}
	}
	// 默认尝试 root
	return "root"
}

// getNetworkMode 获取网络模式
func getNetworkMode(c *pb.Container) string {
	for _, net := range c.Networks {
		if net.NetworkName == "host" {
			return "host"
		}
	}
	return "bridge"
}

// analyzeMounts 分析挂载点，标记危险挂载
func analyzeMounts(mounts []*pb.MountPoint) []*pb.ContainerMount {
	var result []*pb.ContainerMount
	for _, m := range mounts {
		mountInfo := &pb.ContainerMount{
			Source:      m.Source,
			Destination: m.Destination,
			Type:        m.Type,
			Driver:      m.Driver,
			IsDangerous: false,
		}

		// 检查是否危险
		for dangerousPath, reason := range DangerousMounts {
			if strings.HasPrefix(m.Destination, dangerousPath) ||
				strings.HasPrefix(m.Source, dangerousPath) {
				mountInfo.IsDangerous = true
				mountInfo.RiskReason = reason
				break
			}
		}

		result = append(result, mountInfo)
	}
	return result
}

// analyzeCapabilities 分析容器的 capabilities，区分默认开启和手动开启
func analyzeCapabilities(capAdd, capDrop []string) []*pb.ContainerCapability {
	var result []*pb.ContainerCapability

	// 如果 capAdd 和 capDrop 都是 nil 或空，说明没有配置，返回空
	capAddIsNil := capAdd == nil || len(capAdd) == 0
	capDropIsNil := capDrop == nil || len(capDrop) == 0

	// 将 capAdd 转为 map 方便查找
	capAddMap := make(map[string]bool)
	for _, c := range capAdd {
		capAddMap[strings.ToUpper(c)] = true
	}

	// 将 capDrop 转为 map 方便查找
	capDropMap := make(map[string]bool)
	for _, c := range capDrop {
		capDropMap[strings.ToUpper(c)] = true
	}

	// 如果 capAdd 和 capDrop 都是 nil，说明没有配置信息，返回空
	// 只有当有明确的 capAdd 或 capDrop 配置时才进行分析
	if capAddIsNil && capDropIsNil {
		return result
	}

	// 构建所有可能开启的 capabilities
	// 实际有效 = 默认开启 - capDrop + capAdd
	allCaps := make(map[string]bool)

	// 添加默认 capabilities
	for _, c := range DefaultCapabilities {
		allCaps[strings.ToUpper(c)] = true
	}

	// 移除被 drop 的
	for _, c := range capDrop {
		delete(allCaps, strings.ToUpper(c))
	}

	// 添加手动添加的
	for _, c := range capAdd {
		allCaps[strings.ToUpper(c)] = true
	}

	// 生成结果
	for cap := range allCaps {
		isDefault := false
		// 检查是否是默认开启的（在 DefaultCapabilities 中且没有被 drop）
		for _, dc := range DefaultCapabilities {
			if strings.ToUpper(dc) == cap && !capDropMap[strings.ToUpper(dc)] {
				isDefault = true
				break
			}
		}

		// 检查是否手动添加的
		isManualAdd := capAddMap[cap]

		// 如果是手动添加的，则不是默认的
		if isManualAdd {
			isDefault = false
		}

		capInfo := &pb.ContainerCapability{
			Name:        cap,
			IsDangerous: false,
			IsDefault:   isDefault,
		}

		if desc, ok := DangerousCapabilities[cap]; ok {
			capInfo.Description = desc
			capInfo.IsDangerous = true
			capInfo.RiskReason = desc
		}

		result = append(result, capInfo)
	}

	return result
}

// evaluateContainerRisk 初步评估容器风险
func evaluateContainerRisk(info *pb.ContainerEscapeInfo) (bool, string) {
	var risks []string

	if info.Privileged {
		risks = append(risks, "容器以 privileged 模式运行")
	}
	if info.HostPid {
		risks = append(risks, "共享宿主进程命名空间")
	}
	if info.HostNetwork {
		risks = append(risks, "共享宿主网络命名空间")
	}
	if info.HostIpc {
		risks = append(risks, "共享宿主 IPC 命名空间")
	}
	if info.HostUts {
		risks = append(risks, "共享宿主 UTS 命名空间")
	}

	// 检查危险挂载
	dangerousCount := 0
	for _, m := range info.Mounts {
		if m.IsDangerous {
			dangerousCount++
			risks = append(risks, "危险挂载: "+m.Destination)
		}
	}
	if dangerousCount > 0 {
		risks = append(risks, "共发现危险挂载")
	}

	// 检查危险 capabilities
	dangerousCaps := 0
	for _, c := range info.Capabilities {
		if c.IsDangerous {
			dangerousCaps++
		}
	}
	if dangerousCaps > 3 {
		risks = append(risks, "容器具有多个危险 capabilities")
	}

	if len(risks) == 0 {
		return false, "未发现明显逃逸风险"
	}

	summary := strings.Join(risks[:min(3, len(risks))], "; ")
	if len(risks) > 3 {
		summary += "..."
	}

	// 有任何风险项就判定为高危
	return len(risks) >= 1, summary
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}