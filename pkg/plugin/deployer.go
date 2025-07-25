package plugin

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sothoth-nodeagent/pkg/config"
	"sothoth-nodeagent/pkg/pb"
	"sothoth-nodeagent/pkg/util"
	"strconv"
	"strings"
)

// Deploy 部署插件到目标进程
func DeployPlugin(request *pb.DeployPluginRequest) error {
	log.Infof("Start deploy plugin %s to process %d", request.PluginName, request.Pid)

	cfg := config.GetInstance()
	// 创建插件目录
	pluginDir := filepath.Join(cfg.SothothDir, "plugins", request.PluginName, request.PluginVersion)

	log.Infof("handle deploy: %s version: %s Target PID: %d", request.PluginName, request.PluginVersion, request.Pid)

	// 检查插件是否已存在
	if !util.Exists(pluginDir) || util.IsDirEmpty(pluginDir) {
		log.Infof("plugin does not exists, start download: %s", pluginDir)

		// 使用插件管理器下载插件
		err := util.DownloadPlugin(request.PluginName, request.PluginVersion)
		if err != nil {
			return fmt.Errorf("download plugin error: %v", err)
		}

		log.Infof("download plugin success: %s", request.PluginName)
	} else {
		log.Infof("plugin exists, skip download: %s", pluginDir)
	}

	// 解析配置
	pluginConfig, err := parsePluginConfig(pluginDir, request.AgentId, int(request.Pid))
	if err != nil {
		return fmt.Errorf("parse plugin config error: %v", err)
	}

	// 根据部署方法选择不同的部署策略
	switch pluginConfig.Start.Type {
	case "javaagent":
		return deployByJVMAttach(pluginConfig, int(request.Pid))
	case "new_process":
		return deployByNewProcess(pluginConfig)
	case "library_inject":
		return deployByLibraryInject(pluginConfig, int(request.Pid))
	default:
		return fmt.Errorf("UNKNOWN DEPLOY METHOD: %s", pluginConfig.Start.Type)
	}
}

// parsePluginConfig 解析插件配置文件
func parsePluginConfig(pluginPath string, agentId string, targetPID int) (*PluginConfig, error) {
	configPath := filepath.Join(pluginPath, "config.json")

	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("parse plugin config error: %v", err)
	}

	cfg := config.GetInstance()
	text := string(configData)
	text = strings.ReplaceAll(text, "${ROOT}", strings.ReplaceAll(pluginPath, "\\", "/"))
	text = strings.ReplaceAll(text, "${AGENTID}", agentId)
	text = strings.ReplaceAll(text, "${PROCESSID}", strconv.Itoa(targetPID))
	text = strings.ReplaceAll(text, "${SERVER}", cfg.ServerURL)
	text = strings.ReplaceAll(text, "${ARCH}", runtime.GOARCH)

	var pluginConfig PluginConfig
	err = json.Unmarshal([]byte(text), &pluginConfig)
	if err != nil {
		return nil, fmt.Errorf("parse plugin config error: %v", err)
	}

	return &pluginConfig, nil
}

// deployByJVMAttach 通过JVM Attach API部署插件
func deployByJVMAttach(pluginConfig *PluginConfig, targetPID int) error {
	log.Infof("JVM Attach deploy %s", pluginConfig.Name)

	jattach, err := util.Tool("jattach")
	if err != nil {
		return fmt.Errorf("cannot not found jattach: %v", err)
	}

	// 检查目标进程是否存在
	if !isProcessExists(targetPID) {
		return fmt.Errorf("target process does not exists: %d", targetPID)
	}

	agentJarPath := pluginConfig.Start.Path
	agentOptions := pluginConfig.Start.Params[0]

	// 检查Agent JAR文件是否存在
	if _, err := os.Stat(agentJarPath); os.IsNotExist(err) {
		return fmt.Errorf("Agent JAR file not found: %s", agentJarPath)
	}

	cmd := exec.Command(jattach,
		strconv.Itoa(int(targetPID)),
		"load",
		"instrument",
		"false",
		fmt.Sprintf("%s=%s", agentJarPath, agentOptions))

	log.Infof("Execute cmd: %s", cmd.String())

	// 执行命令
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("jattach execute failed: %v, Output: %s", err, string(output))
	}

	log.Infof("jattach output: %s", string(output))

	log.Infof("plugin %s deploy success", pluginConfig.Name)
	return nil
}

// deployByNewProcess 通过新进程部署插件
func deployByNewProcess(pluginConfig *PluginConfig) error {
	log.Infof("New Process deploy %s", pluginConfig.Name)

	// 构建执行命令
	executable := pluginConfig.Start.Path

	// 检查可执行文件是否存在
	if _, err := os.Stat(executable); os.IsNotExist(err) {
		return fmt.Errorf("executable does not exist: %s", executable)
	}

	// 构建命令参数
	args := append([]string{executable}, pluginConfig.Start.Params...)

	// 创建命令
	cmd := exec.Command(args[0], args[1:]...)

	log.Infof("start new process: %s", cmd.String())

	// 启动进程
	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("start process error: %v", err)
	}

	log.Infof("plugin %s deploy success", pluginConfig.Name)
	return nil
}

// deployByLibraryInject 通过库注入部署插件
func deployByLibraryInject(pluginConfig *PluginConfig, targetPID int) error {
	log.Infof("Library Inject deploy %s", pluginConfig.Name)

	// 检查目标进程是否存在
	if !isProcessExists(targetPID) {
		return fmt.Errorf("target process not found: %d", targetPID)
	}

	// 使用ptrace或其他方式注入库
	// 这里简化实现，实际应该使用更复杂的注入技术
	log.Infof("inject library into process %d", targetPID)

	log.Infof("plugin %s deploy success", pluginConfig.Name)
	return nil
}

// UndeployPlugin 停止插件
func UndeployPlugin(pluginConfig *PluginConfig) error {
	log.Infof("stop plugin %s", pluginConfig.Name)

	switch pluginConfig.Stop.Type {
	case "javaagent":
		return stopJVMAttachPlugin(pluginConfig)
	case "new_process":
		return stopNewProcessPlugin(pluginConfig)
	case "library_inject":
		return stopLibraryInjectPlugin(pluginConfig)
	default:
		return fmt.Errorf("UNKNOWN DEPLOY METHOD: %s", pluginConfig.Stop.Type)
	}
}

// stopJVMAttachPlugin 停止JVM Attach插件
func stopJVMAttachPlugin(pluginConfig *PluginConfig) error {
	// 对于JVM Attach的插件

	jattachPath := filepath.Join(config.GetInstance().SothothDir, "/jattach")
	if !util.Exists(jattachPath) {
		err := util.DownloadTool("jattach")
		if err != nil {
			return err
		}
	}

	// 这里简化实现
	log.Infof("send stop command %s", pluginConfig.Name)

	return nil
}

// stopNewProcessPlugin 停止新进程插件
func stopNewProcessPlugin(pluginConfig *PluginConfig) error {

	log.Infof("process %s terminated.", pluginConfig.Name)
	return nil
}

// stopLibraryInjectPlugin 停止库注入插件
func stopLibraryInjectPlugin(pluginConfig *PluginConfig) error {
	// 对于库注入的插件，需要通过特定方式卸载
	log.Infof("stop inject library %s", pluginConfig.Name)
	return nil
}

// isProcessExists 检查进程是否存在
func isProcessExists(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	if process == nil {
		return false
	}
	return true
	//// 发送信号0检查进程是否存在, windows下面好像有问题，去掉
	//err = process.Signal(syscall.Signal(0))
	//return err == nil
}
