package escape

import (
	"gaiasec-nodeagent/pkg/pb"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// AnalyzePrivilegeEscalation 分析本地提权风险
func AnalyzePrivilegeEscalation() (*pb.PrivilegeEscalationInfo, error) {
	info := &pb.PrivilegeEscalationInfo{
		NodeName: getNodeName(),
	}

	// 获取当前用户信息
	if err := fillUserInfo(info); err != nil {
		return nil, err
	}

	// 查找 SUID/SGID 可执行文件
	info.SuidBinaries = findSuidBinaries()

	// 分析 sudo 配置
	info.SudoConfig = analyzeSudoConfig()

	// 获取当前进程 capabilities
	info.Capabilities = getCurrentCapabilities()

	// 检查可写入的敏感文件
	info.WritableSensitiveFiles = checkWritableSensitiveFiles()

	// 检查敏感环境变量
	info.SensitiveEnvVars = checkSensitiveEnvVars()

	// 初步风险评估
	info.HasHighRisk, info.RiskSummary = evaluatePrivilegeRisk(info)

	return info, nil
}

// getNodeName 获取节点名称
func getNodeName() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

// fillUserInfo 填充当前用户信息
func fillUserInfo(info *pb.PrivilegeEscalationInfo) error {
	current, err := user.Current()
	if err != nil {
		return err
	}

	info.CurrentUser = current.Username
	info.CurrentUid = 0
	info.CurrentGid = 0

	// 解析 UID
	if uid, err := strconv.Atoi(current.Uid); err == nil {
		info.CurrentUid = int32(uid)
	}
	if gid, err := strconv.Atoi(current.Gid); err == nil {
		info.CurrentGid = int32(gid)
	}

	// 获取用户组
	groups, err := current.GroupIds()
	if err == nil {
		info.Groups = groups
	}

	return nil
}

// findSuidBinaries 查找 SUID/SGID 可执行文件
func findSuidBinaries() []*pb.SuidBinary {
	var results []*pb.SuidBinary

	// 需要搜索的目录
	searchPaths := []string{
		"/bin", "/sbin", "/usr/bin", "/usr/sbin",
		"/usr/local/bin", "/opt",
	}

	// 已检查的路径，防止重复
	checked := make(map[string]bool)

	for _, searchPath := range searchPaths {
		filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !info.Mode().IsRegular() {
				return nil
			}

			// 检查 SUID 位
			mode := info.Mode()
			isSuid := mode&os.ModeSetuid != 0
			isSgid := mode&os.ModeSetgid != 0

			if isSuid || isSgid {
				// 避免重复
				if checked[path] {
					return nil
				}
				checked[path] = true

				suidInfo := &pb.SuidBinary{
					Path:        path,
					Owner:       strconv.FormatUint(uint64(info.Sys().(*syscall.Stat_t).Uid), 10),
					Group:       strconv.FormatUint(uint64(info.Sys().(*syscall.Stat_t).Gid), 10),
					IsSuid:      isSuid,
					IsSgid:      isSgid,
					IsDangerous: false,
				}

				// 检查是否危险
				baseName := filepath.Base(path)
				if reason, ok := DangerousSuidBinaries["/"+baseName]; ok {
					suidInfo.IsDangerous = true
					suidInfo.RiskReason = reason
				}

				results = append(results, suidInfo)
			}

			return nil
		})
	}

	return results
}

// analyzeSudoConfig 分析 sudo 配置
func analyzeSudoConfig() []*pb.SudoConfig {
	var results []*pb.SudoConfig

	// 读取 /etc/sudoers
	sudoersPaths := []string{"/etc/sudoers", "/etc/sudoers.d/"}

	for _, sudoersPath := range sudoersPaths {
		filepath.Walk(sudoersPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				return nil
			}

			// 读取 sudoers 文件内容
			content, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				// 跳过注释和空行
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}

				// 解析 sudo 配置行
				parts := strings.Fields(line)
				if len(parts) >= 3 {
					sudoConfig := &pb.SudoConfig{
						User: parts[0],
						Host: parts[1],
						Command: strings.Join(parts[2:], " "),
						IsPasswordless: !strings.Contains(line, "password"),
					}
					results = append(results, sudoConfig)
				}
			}

			return nil
		})
	}

	return results
}

// getCurrentCapabilities 获取当前进程的 capabilities
func getCurrentCapabilities() []*pb.ContainerCapability {
	// Linux capabilities 可以通过 /proc/self/status 获取
	// 这里简化处理，返回常见的 capabilities

	capContent, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return nil
	}

	var caps []string
	for _, line := range strings.Split(string(capContent), "\n") {
		if strings.HasPrefix(line, "CapEff:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				caps = strings.Split(parts[1], ",")
			}
			break
		}
	}

	// 映射十六进制到能力名
	capNames := map[string]string{
		"00000000": "CAP_CHOWN", "00000001": "CAP_DAC_OVERRIDE",
		"00000002": "CAP_DAC_READ_SEARCH", "00000003": "CAP_FOWNER",
		"00000004": "CAP_FSETID", "00000005": "CAP_KILL",
		"00000006": "CAP_SETGID", "00000007": "CAP_SETUID",
		"00000008": "CAP_SETPCAP", "00000009": "CAP_LINUX_IMMUTABLE",
		"0000000a": "CAP_NET_BIND_SERVICE", "0000000b": "CAP_NET_BROADCAST",
		"0000000c": "CAP_NET_ADMIN", "0000000d": "CAP_NET_RAW",
		"0000000e": "CAP_IPC_LOCK", "0000000f": "CAP_IPC_OWNER",
		"00000010": "CAP_SYS_MODULE", "00000011": "CAP_SYS_RAWIO",
		"00000012": "CAP_SYS_CHROOT", "00000013": "CAP_SYS_PTRACE",
		"00000014": "CAP_SYS_PACCT", "00000015": "CAP_SYS_ADMIN",
		"00000016": "CAP_SYS_BOOT", "00000017": "CAP_SYS_NICE",
		"00000018": "CAP_SYS_RESOURCE", "00000019": "CAP_SYS_TIME",
		"0000001a": "CAP_SYS_TTY_CONFIG", "0000001b": "CAP_MKNOD",
		"0000001c": "CAP_LEASE", "0000001d": "CAP_AUDIT_WRITE",
		"0000001e": "CAP_AUDIT_CONTROL", "0000001f": "CAP_SETFCAP",
	}

	var results []*pb.ContainerCapability
	for _, capHex := range caps {
		if name, ok := capNames[strings.ToUpper(capHex)]; ok {
			capInfo := &pb.ContainerCapability{
				Name: name,
				IsDangerous: false,
			}
			if desc, ok := DangerousCapabilities[name]; ok {
				capInfo.Description = desc
				capInfo.IsDangerous = true
				capInfo.RiskReason = desc
			}
			results = append(results, capInfo)
		}
	}

	return results
}

// checkWritableSensitiveFiles 检查可写入的敏感文件
func checkWritableSensitiveFiles() []string {
	var results []string

	for _, filePath := range SensitiveWritableFiles {
		// 支持通配符
		if strings.Contains(filePath, "*") {
			continue // 简化处理，跳过通配符
		}

		// 检查文件是否存在且可写
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			continue
		}
		_ = fileInfo // 避免未使用警告

		// 检查写权限 (简化判断)
		file, err := os.OpenFile(filePath, os.O_WRONLY, 0)
		if err == nil {
			file.Close()
			results = append(results, filePath)
		}
	}

	return results
}

// checkSensitiveEnvVars 检查敏感环境变量
func checkSensitiveEnvVars() []string {
	sensitiveVars := []string{
		"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY",
		"AZURE_CLIENT_SECRET", "GCP_SERVICE_ACCOUNT_KEY",
		"DATABASE_URL", "DB_PASSWORD", "DB_USER",
		"SSH_PRIVATE_KEY", "PRIVATE_KEY",
		"TOKEN", "API_KEY", "SECRET",
		"JWT_SECRET", "SESSION_SECRET",
	}

	var results []string
	for _, varName := range sensitiveVars {
		if value := os.Getenv(varName); value != "" {
			// 不输出实际值，只标记
			results = append(results, varName)
		}
	}

	return results
}

// evaluatePrivilegeRisk 初步评估提权风险
func evaluatePrivilegeRisk(info *pb.PrivilegeEscalationInfo) (bool, string) {
	var risks []string

	// 检查是否为 root 用户
	if info.CurrentUid == 0 {
		return false, "当前已是 root 用户"
	}

	// 检查危险 SUID 文件
	dangerousSuid := 0
	for _, sb := range info.SuidBinaries {
		if sb.IsDangerous {
			dangerousSuid++
		}
	}
	if dangerousSuid > 0 {
		risks = append(risks, "存在危险 SUID 文件")
	}

	// 检查无密码 sudo 配置
	passwordlessSudo := 0
	for _, sc := range info.SudoConfig {
		if sc.IsPasswordless {
			passwordlessSudo++
		}
	}
	if passwordlessSudo > 0 {
		risks = append(risks, "存在无密码 sudo 配置")
	}

	// 检查危险 capabilities
	dangerousCaps := 0
	for _, c := range info.Capabilities {
		if c.IsDangerous {
			dangerousCaps++
		}
	}
	if dangerousCaps > 0 {
		risks = append(risks, "具有危险 capabilities")
	}

	// 检查可写敏感文件
	if len(info.WritableSensitiveFiles) > 0 {
		risks = append(risks, "存在可写入的敏感文件")
	}

	if len(risks) == 0 {
		return false, "未发现明显提权路径"
	}

	summary := strings.Join(risks[:min(3, len(risks))], "; ")
	return true, summary
}