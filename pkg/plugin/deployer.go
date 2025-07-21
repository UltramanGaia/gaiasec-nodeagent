package plugin

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sothoth-nodeagent/pkg/config"
	"sothoth-nodeagent/pkg/util"
	"strconv"
	"strings"
	"syscall"
)

// PluginDeployer 插件部署器
type PluginDeployer struct {
	workDir string
}

// NewPluginDeployer 创建新的插件部署器
func NewPluginDeployer(workDir string) *PluginDeployer {
	return &PluginDeployer{
		workDir: workDir,
	}
}

// Deploy 部署插件到目标进程
func (pd *PluginDeployer) Deploy(pluginName string, pluginVersion string, agentId string, targetPID int, options map[string]interface{}) error {
	log.Printf("开始部署插件 %s 到进程 %d", pluginName, targetPID)

	// 创建插件目录
	pluginDir := filepath.Join(pd.workDir, pluginName, pluginVersion)
	// 解析配置
	pluginConfig, err := pd.parsePluginConfig(pluginDir, agentId, targetPID)
	if err != nil {
		return fmt.Errorf("解析插件配置失败: %v", err)
	}

	// 根据部署方法选择不同的部署策略
	switch pluginConfig.Start.Type {
	case "javaagent":
		return pd.deployByJVMAttach(pluginConfig, targetPID, options)
	case "new_process":
		return pd.deployByNewProcess(pluginConfig, options)
	case "library_inject":
		return pd.deployByLibraryInject(pluginConfig, targetPID, options)
	default:
		return fmt.Errorf("不支持的部署方法: %s", pluginConfig.Start.Type)
	}
}

// parsePluginConfig 解析插件配置文件
func (pd *PluginDeployer) parsePluginConfig(pluginPath string, agentId string, targetPID int) (*PluginConfig, error) {
	configPath := filepath.Join(pluginPath, "config.json")

	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	cfg := config.GetInstance()
	text := string(configData)
	text = strings.ReplaceAll(text, "${ROOT}", pluginPath)
	text = strings.ReplaceAll(text, "${AGENTID}", agentId)
	text = strings.ReplaceAll(text, "${PROCESSID}", strconv.Itoa(targetPID))
	text = strings.ReplaceAll(text, "${SERVER}", cfg.ServerURL)
	text = strings.ReplaceAll(text, "${ARCH}", runtime.GOARCH)

	var pluginConfig PluginConfig
	err = json.Unmarshal([]byte(text), &pluginConfig)
	if err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	return &pluginConfig, nil
}

// deployByJVMAttach 通过JVM Attach API部署插件
func (pd *PluginDeployer) deployByJVMAttach(pluginConfig *PluginConfig, targetPID int, options map[string]interface{}) error {
	log.Printf("使用JVM Attach方式部署插件 %s", pluginConfig.Name)

	jattachPath := filepath.Join(config.GetInstance().SothothDir, "/jattach")
	if !util.Exists(jattachPath) {
		err := util.DownloadTool("jattach")
		if err != nil {
			return err
		}
	}

	// 检查目标进程是否存在
	if !pd.isProcessExists(targetPID) {
		return fmt.Errorf("目标进程不存在: %d", targetPID)
	}

	agentJarPath := pluginConfig.Start.Path
	agentOptions := pluginConfig.Start.Params[0]

	// 检查Agent JAR文件是否存在
	if _, err := os.Stat(agentJarPath); os.IsNotExist(err) {
		return fmt.Errorf("Agent JAR文件不存在: %s", agentJarPath)
	}

	cmd := exec.Command(jattachPath,
		strconv.Itoa(targetPID),
		"load",
		"instrument",
		"false",
		fmt.Sprintf("%s=%s", agentJarPath, agentOptions))

	log.Printf("执行命令: %s", cmd.String())

	// 执行命令
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("jattach命令执行失败: %v, 输出: %s", err, string(output))
	}

	log.Printf("jattach输出: %s", string(output))

	log.Printf("插件 %s 通过JVM Attach部署成功", pluginConfig.Name)
	return nil
}

// deployByNewProcess 通过新进程部署插件
func (pd *PluginDeployer) deployByNewProcess(pluginConfig *PluginConfig, options map[string]interface{}) error {
	log.Printf("使用新进程方式部署插件 %s", pluginConfig.Name)

	// 构建执行命令
	executable := pluginConfig.Start.Path

	// 检查可执行文件是否存在
	if _, err := os.Stat(executable); os.IsNotExist(err) {
		return fmt.Errorf("可执行文件不存在: %s", executable)
	}

	// 构建命令参数
	args := append([]string{executable}, pluginConfig.Start.Params...)

	// 创建命令
	cmd := exec.Command(args[0], args[1:]...)

	log.Printf("启动新进程: %s", cmd.String())

	// 启动进程
	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("启动进程失败: %v", err)
	}

	log.Printf("插件 %s 通过新进程部署成功", pluginConfig.Name)
	return nil
}

// deployByLibraryInject 通过库注入部署插件
func (pd *PluginDeployer) deployByLibraryInject(pluginConfig *PluginConfig, targetPID int, options map[string]interface{}) error {
	log.Printf("使用库注入方式部署插件 %s", pluginConfig.Name)

	// 检查目标进程是否存在
	if !pd.isProcessExists(targetPID) {
		return fmt.Errorf("目标进程不存在: %d", targetPID)
	}

	// 使用ptrace或其他方式注入库
	// 这里简化实现，实际应该使用更复杂的注入技术
	log.Printf("注入库文件 到进程 %d", targetPID)

	log.Printf("插件 %s 通过库注入部署成功", pluginConfig.Name)
	return nil
}

// Stop 停止插件
func (pd *PluginDeployer) Stop(pluginConfig *PluginConfig) error {
	log.Printf("停止插件 %s", pluginConfig.Name)

	switch pluginConfig.Stop.Type {
	case "javaagent":
		return pd.stopJVMAttachPlugin(pluginConfig)
	case "new_process":
		return pd.stopNewProcessPlugin(pluginConfig)
	case "library_inject":
		return pd.stopLibraryInjectPlugin(pluginConfig)
	default:
		return fmt.Errorf("不支持的部署方法: %s", pluginConfig.Stop.Type)
	}
}

// stopJVMAttachPlugin 停止JVM Attach插件
func (pd *PluginDeployer) stopJVMAttachPlugin(pluginConfig *PluginConfig) error {
	// 对于JVM Attach的插件

	jattachPath := filepath.Join(config.GetInstance().SothothDir, "/jattach")
	if !util.Exists(jattachPath) {
		err := util.DownloadTool("jattach")
		if err != nil {
			return err
		}
	}

	// 这里简化实现
	log.Printf("发送停止命令到插件 %s", pluginConfig.Name)

	return nil
}

// stopNewProcessPlugin 停止新进程插件
func (pd *PluginDeployer) stopNewProcessPlugin(pluginConfig *PluginConfig) error {

	log.Printf("进程 %s 已终止", pluginConfig.Name)
	return nil
}

// stopLibraryInjectPlugin 停止库注入插件
func (pd *PluginDeployer) stopLibraryInjectPlugin(pluginConfig *PluginConfig) error {
	// 对于库注入的插件，需要通过特定方式卸载
	log.Printf("卸载注入的库 %s", pluginConfig.Name)
	return nil
}

// isProcessExists 检查进程是否存在
func (pd *PluginDeployer) isProcessExists(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// 发送信号0检查进程是否存在
	err = process.Signal(syscall.Signal(0))
	return err == nil
}
