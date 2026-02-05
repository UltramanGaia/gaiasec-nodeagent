package util

import (
	"path/filepath"
	"runtime"
	"gaiasec-nodeagent/pkg/config"
)

func Tool(toolName string) (string, error) {
	toolPath := filepath.Join(config.GetInstance().GaiaSecDir, toolName)
	if runtime.GOOS == "windows" {
		toolPath += ".exe"
	}
	if !Exists(toolPath) {
		err := DownloadTool(toolName)
		if err != nil {
			return "", err
		}
	}
	return toolPath, nil
}
