// Package plugin 提供插件管理功能
//
// 该包负责：
// - 插件下载和部署
// - 插件配置解析
// - 插件生命周期管理
// - 插件状态监控
package plugin

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sothoth-nodeagent/pkg/config"
	"sothoth-nodeagent/pkg/util"
	"sync"
)

// PluginManager 插件管理器
type PluginManager struct {
	deployer *PluginDeployer
	mu       sync.RWMutex
	workDir  string
}

type Dependency struct {
}

type CommandParam struct {
	Type   string
	Path   string
	Params []string
}

type PluginConfig struct {
	Name         string
	Version      string
	Dependencies []Dependency
	Type         string
	Start        CommandParam
	Stop         CommandParam
}

// NewPluginManager 创建新的插件管理器
func NewPluginManager(workDir string) *PluginManager {
	pluginDir := filepath.Join(workDir, "plugins")
	os.MkdirAll(pluginDir, 0755)

	return &PluginManager{
		deployer: NewPluginDeployer(pluginDir),
		workDir:  pluginDir,
	}
}

// DeployPlugin 部署插件
func (pm *PluginManager) DeployPlugin(name string, version string, agentId string, targetPID int, options map[string]interface{}) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	log.Printf("开始部署插件: %s 版本: %s", name, version)
	cfg := config.GetInstance()
	pluginDir := filepath.Join(cfg.SothothDir, "plugins", name, version)

	log.Printf("处理插件部署请求: %s 版本: %s 目标PID: %d", name, version, targetPID)

	// 检查插件是否已存在
	if !util.Exists(pluginDir) {
		log.Printf("插件不存在，开始下载: %s", pluginDir)

		// 使用插件管理器下载插件
		err := util.DownloadPlugin(name, version)
		if err != nil {
			return fmt.Errorf("下载插件失败: %v", err)
		}

		log.Printf("插件下载完成: %s", name)
	} else {
		log.Printf("插件已存在，跳过下载: %s", pluginDir)
	}

	// 部署插件
	err := pm.deployer.Deploy(name, version, agentId, targetPID, options)
	if err != nil {
		return fmt.Errorf("部署插件失败: %v", err)
	}

	log.Printf("插件 %s 部署成功", name)
	return nil
}
