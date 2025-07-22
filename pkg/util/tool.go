package util

import (
	"path/filepath"
	"runtime"
	"sothoth-nodeagent/pkg/config"
)

func Tool(toolName string) (string, error) {
	toolPath := filepath.Join(config.GetInstance().SothothDir, toolName)
	if runtime.GOOS == "windows" {
		toolPath += ".exe"
	}
	if !Exists(toolPath) {
		err := DownloadTool("jattach")
		if err != nil {
			return "", err
		}
	}
	return toolPath, nil
}
