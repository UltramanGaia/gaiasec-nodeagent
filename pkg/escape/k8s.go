package escape

import (
	"gaiasec-nodeagent/pkg/pb"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// AnalyzeK8SPrivilegeEscalation 分析 K8s 集群内提权风险
func AnalyzeK8SPrivilegeEscalation() (*pb.K8SPrivilegeEscalationInfo, error) {
	info := &pb.K8SPrivilegeEscalationInfo{
		NodeName: getNodeName(),
	}

	// 填充 K8s 环境信息
	if err := fillK8sEnvInfo(info); err != nil {
		return nil, err
	}

	// 获取 RBAC 信息
	info.ClusterRoleBindings, info.RoleBindings = analyzeRBAC()

	// 获取可访问的 Secrets
	info.AccessibleSecrets = getAccessibleSecrets()

	// 获取 Token 信息
	analyzeTokenInfo(info)

	// 检查敏感操作权限
	info.CanExec, info.CanPortForward, info.CanCreatePod, info.CanDeletePod = checkSensitivePermissions(info.Namespace)

	// 初步风险评估
	info.HasHighRisk, info.RiskSummary = evaluateK8sPrivilegeRisk(info)

	return info, nil
}

// fillK8sEnvInfo 填充 K8s 环境信息
func fillK8sEnvInfo(info *pb.K8SPrivilegeEscalationInfo) error {
	// 尝试从 /var/run/secrets/kubernetes.io/serviceaccount/ 读取信息
	saPath := "/var/run/secrets/kubernetes.io/serviceaccount"

	// 检查是否在 K8s 环境中
	if _, err := os.Stat(saPath); os.IsNotExist(err) {
		info.NodeName = getNodeName()
		info.ClusterName = "not-in-k8s"
		info.Namespace = "default"
		info.ServiceAccount = "default"
		return nil
	}

	// 读取 namespace
	if ns, err := os.ReadFile(filepath.Join(saPath, "namespace")); err == nil {
		info.Namespace = strings.TrimSpace(string(ns))
	} else {
		info.Namespace = "default"
	}

	// 读取 service account 名称
	if sa, err := os.ReadFile(filepath.Join(saPath, "token")); err == nil {
		info.ServiceAccount = "serviceaccount"
		_ = sa // token 内容
	}

	// 获取 cluster name (通过 kubectl 或 API)
	info.ClusterName = getClusterName()

	return nil
}

// getClusterName 获取集群名称
func getClusterName() string {
	// 尝试通过 kubectl 获取
	cmd := exec.Command("kubectl", "config", "current-context")
	output, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(output))
	}

	// 尝试读取 in-cluster 配置
	clusterFile := "/var/run/secrets/kubernetes.io/serviceaccount/../.."
	if _, err := os.Stat(filepath.Join(clusterFile, "cluster_name")); err == nil {
		if data, err := os.ReadFile(filepath.Join(clusterFile, "cluster_name")); err == nil {
			return strings.TrimSpace(string(data))
		}
	}

	return "unknown"
}

// analyzeRBAC 分析 RBAC 权限
func analyzeRBAC() ([]*pb.ClusterRoleBindingInfo, []*pb.ClusterRoleBindingInfo) {
	var clusterRoleBindings []*pb.ClusterRoleBindingInfo
	var roleBindings []*pb.ClusterRoleBindingInfo

	// 获取 ClusterRoleBindings
	cmd := exec.Command("kubectl", "get", "clusterrolebindings", "-o", "json")
	output, err := cmd.Output()
	if err != nil {
		// 可能没有 kubectl 或权限，尝试其他方式
		return getClusterRoleBindingsFromAPI()
	}

	// 解析 JSON 输出 (简化处理)
	// 实际实现应该用 JSON 解析
	clusterRoleBindings = parseClusterRoleBindings(string(output))

	// 获取 RoleBindings (当前 namespace)
	cmd = exec.Command("kubectl", "get", "rolebindings", "-n", "default", "-o", "json")
	output, err = cmd.Output()
	if err == nil {
		roleBindings = parseRoleBindings(string(output))
	}

	return clusterRoleBindings, roleBindings
}

// getClusterRoleBindingsFromAPI 通过 API 获取 ClusterRoleBindings
func getClusterRoleBindingsFromAPI() ([]*pb.ClusterRoleBindingInfo, []*pb.ClusterRoleBindingInfo) {
	// 简化：使用 curl 直接调用 K8s API
	var results []*pb.ClusterRoleBindingInfo

	// 获取当前 service account 的 token
	tokenPath := "/var/run/secrets/kubernetes.io/serviceaccount/token"
	token, err := os.ReadFile(tokenPath)
	if err != nil {
		return results, nil
	}

	// 通过 K8s API 获取当前 pod 的信息
	cmd := exec.Command("curl", "-sk",
		"https://kubernetes.default.svc/apis/rbac.authorization.k8s.io/v1/clusterrolebindings",
		"-H", "Authorization: Bearer "+string(token))
	output, err := cmd.Output()
	if err != nil {
		return results, nil
	}

	return parseClusterRoleBindings(string(output)), nil
}

// parseClusterRoleBindings 解析 ClusterRoleBindings JSON
func parseClusterRoleBindings(jsonOutput string) []*pb.ClusterRoleBindingInfo {
	var results []*pb.ClusterRoleBindingInfo

	// 简化解析 - 实际应该用 JSON 解析库
	// 这里检测危险角色名称
	dangerousRoles := []string{
		"cluster-admin", "admin", "edit", "root",
		"system:master", "system:kube-controller-manager",
		"system:kube-scheduler", "kubelet-api",
	}

	lines := strings.Split(jsonOutput, "\n")
	for _, line := range lines {
		if strings.Contains(line, `"name"`) {
			// 提取 name
			parts := strings.Split(line, `"name"`)
			if len(parts) >= 2 {
				namePart := strings.Split(parts[1], `"`)[1]
				binding := &pb.ClusterRoleBindingInfo{
					Name:          namePart,
					CanPrivilegeEsc: false,
				}

				// 检查是否危险
				for _, role := range dangerousRoles {
					if strings.Contains(strings.ToLower(namePart), role) {
						binding.CanPrivilegeEsc = true
						binding.RiskReason = "具有危险角色: " + role
						break
					}
				}
				results = append(results, binding)
			}
		}
	}

	return results
}

// parseRoleBindings 解析 RoleBindings JSON
func parseRoleBindings(jsonOutput string) []*pb.ClusterRoleBindingInfo {
	var results []*pb.ClusterRoleBindingInfo

	// 简化解析
	dangerousRoles := []string{"admin", "edit", "system:editor"}

	lines := strings.Split(jsonOutput, "\n")
	for _, line := range lines {
		if strings.Contains(line, `"name"`) {
			parts := strings.Split(line, `"name"`)
			if len(parts) >= 2 {
				namePart := strings.Split(parts[1], `"`)[1]
				binding := &pb.ClusterRoleBindingInfo{
					Name:          namePart,
					CanPrivilegeEsc: false,
				}

				for _, role := range dangerousRoles {
					if strings.Contains(strings.ToLower(namePart), role) {
						binding.CanPrivilegeEsc = true
						binding.RiskReason = "具有危险角色: " + role
						break
					}
				}
				results = append(results, binding)
			}
		}
	}

	return results
}

// getAccessibleSecrets 获取可访问的 Secrets
func getAccessibleSecrets() []*pb.AccessibleSecret {
	var results []*pb.AccessibleSecret

	// 获取可访问的 secrets
	cmd := exec.Command("kubectl", "get", "secrets", "-o", "json")
	output, err := cmd.Output()
	if err != nil {
		return results
	}

	// 简化解析
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, `"name"`) && strings.Contains(line, `"type"`) {
			parts := strings.Split(line, `"name"`)
			if len(parts) >= 2 {
				name := strings.Split(parts[1], `"`)[1]
				secret := &pb.AccessibleSecret{
					Name:        name,
					Namespace:   "default",
					CanRead:     true, // 简化：假设有读取权限
				}
				results = append(results, secret)
			}
		}
	}

	return results
}

// analyzeTokenInfo 分析 Token 信息
func analyzeTokenInfo(info *pb.K8SPrivilegeEscalationInfo) {
	tokenPath := "/var/run/secrets/kubernetes.io/serviceaccount/token"
	tokenFilePath := "/var/run/secrets/kubernetes.io/serviceaccount/token"

	info.TokenFile = tokenFilePath

	// 尝试读取 token 的 jwt 信息
	// 简化处理：检查 token 文件是否存在
	if _, err := os.Stat(tokenPath); err == nil {
		// token 存在，尝试解析过期时间
		// 实际应该解析 JWT
		info.TokenExpiration = -1 // 未知
	}

	// 检查是否有 ca.crt 来确认是 K8s 环境
	if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"); err == nil {
		// 在 K8s 环境中
	}
}

// checkSensitivePermissions 检查敏感操作权限
func checkSensitivePermissions(namespace string) (bool, bool, bool, bool) {
	canExec := false
	canPortForward := false
	canCreatePod := false
	canDeletePod := false

	// 检查是否可以执行命令
	cmd := exec.Command("kubectl", "auth", "can-i", "exec", "--namespace", namespace)
	if execCmd(cmd) {
		canExec = true
	}

	// 检查是否可以 port-forward
	cmd = exec.Command("kubectl", "auth", "can-i", "port-forward", "--namespace", namespace)
	if execCmd(cmd) {
		canPortForward = true
	}

	// 检查是否可以创建 Pod
	cmd = exec.Command("kubectl", "auth", "can-i", "create", "pods", "--namespace", namespace)
	if execCmd(cmd) {
		canCreatePod = true
	}

	// 检查是否可以删除 Pod
	cmd = exec.Command("kubectl", "auth", "can-i", "delete", "pods", "--namespace", namespace)
	if execCmd(cmd) {
		canDeletePod = true
	}

	return canExec, canPortForward, canCreatePod, canDeletePod
}

// execCmd 执行命令并返回是否成功
func execCmd(cmd *exec.Cmd) bool {
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "yes"
}

// evaluateK8sPrivilegeRisk 初步评估 K8s 提权风险
func evaluateK8sPrivilegeRisk(info *pb.K8SPrivilegeEscalationInfo) (bool, string) {
	var risks []string

	// 检查 cluster-admin 权限
	for _, binding := range info.ClusterRoleBindings {
		if binding.CanPrivilegeEsc {
			risks = append(risks, "具有 ClusterRoleBinding: "+binding.Name)
		}
	}

	// 检查可访问的 secrets
	if len(info.AccessibleSecrets) > 0 {
		risks = append(risks, "可访问 "+string(rune(len(info.AccessibleSecrets)))+" 个 secrets")
	}

	// 检查 token 过期时间
	if info.TokenExpiration == -1 {
		risks = append(risks, "无法确定 token 过期时间")
	}

	// 检查敏感操作权限
	if info.CanExec {
		risks = append(risks, "具有 exec 权限，可执行命令")
	}
	if info.CanCreatePod {
		risks = append(risks, "具有创建 Pod 权限，可用于提权")
	}

	if len(risks) == 0 {
		return false, "未发现明显提权路径"
	}

	summary := strings.Join(risks[:min(3, len(risks))], "; ")
	return true, summary
}