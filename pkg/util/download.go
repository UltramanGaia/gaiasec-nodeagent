package util

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sothoth-nodeagent/pkg/config"
	"strings"
)

// DownloadPlugin 下载并解压插件
func DownloadPlugin(name, version string) error {
	cfg := config.GetInstance()
	log.Printf("开始下载插件: %s 版本: %s", name, version)
	// 构建下载URL
	downloadURL := fmt.Sprintf("/plugins/%s/%s.zip", name, version)

	// 创建插件目录
	pluginDir := filepath.Join(cfg.SothothDir, "plugins", name, version)
	err := os.MkdirAll(pluginDir, 0755)
	if err != nil {
		return fmt.Errorf("创建插件目录失败: %v", err)
	}

	// 下载插件文件
	zipPath := filepath.Join(pluginDir, "..", fmt.Sprintf("%s-%s.zip", name, version))
	err = downloadFile(downloadURL, zipPath)
	if err != nil {
		return fmt.Errorf("下载插件文件失败: %v", err)
	}

	// 解压插件文件
	err = extractZip(zipPath, pluginDir)
	if err != nil {
		return fmt.Errorf("解压插件文件失败: %v", err)
	}

	// 删除Zip文件
	os.Remove(zipPath)

	log.Printf("插件 %s 下载并解压完成", name)
	return nil
}

// DownloadTool 下载工具
func DownloadTool(name string) error {
	cfg := config.GetInstance()
	log.Printf("开始下载工具: %s", name)
	// 构建下载URL
	downloadURL := fmt.Sprintf("/plugins/nodeagent/%s-%s-%s", name, runtime.GOOS, runtime.GOARCH)

	// 下载插件文件
	toolPath := filepath.Join(cfg.SothothDir, name)
	err := downloadFile(downloadURL, toolPath)
	if err != nil {
		return fmt.Errorf("下载工具失败: %v", err)
	}

	os.Chmod(toolPath, 0755)

	log.Printf("工具 %s 下载完成", name)
	return nil
}

// downloadFile 下载文件
func downloadFile(url, filepath string) error {
	// 处理相对URL
	if strings.HasPrefix(url, "/") {
		// 这里应该从配置中获取服务器地址，暂时硬编码
		url = "http://" + config.GetInstance().ServerURL + url
	}

	log.Printf("下载文件: %s", url)

	// 创建HTTP请求
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP响应错误: %d %s", resp.StatusCode, resp.Status)
	}

	// 创建目标文件
	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %v", err)
	}
	defer out.Close()

	// 复制数据
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}

	log.Printf("文件下载完成: %s", filepath)
	return nil
}

// extractZip 解压ZIP文件
func extractZip(src, dest string) error {
	log.Printf("解压文件: %s 到 %s", src, dest)

	// 打开ZIP文件
	r, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("打开ZIP文件失败: %v", err)
	}
	defer r.Close()

	// 创建目标目录
	err = os.MkdirAll(dest, 0755)
	if err != nil {
		return fmt.Errorf("创建目标目录失败: %v", err)
	}

	// 解压文件
	for _, f := range r.File {
		err := extractFile(f, dest)
		if err != nil {
			return fmt.Errorf("解压文件 %s 失败: %v", f.Name, err)
		}
	}

	log.Printf("ZIP文件解压完成")
	return nil
}

// extractFile 解压单个文件
func extractFile(f *zip.File, destDir string) error {
	// 构建目标路径
	path := filepath.Join(destDir, f.Name)

	// 检查路径安全性（防止目录遍历攻击）
	if !strings.HasPrefix(path, filepath.Clean(destDir)+string(os.PathSeparator)) {
		return fmt.Errorf("无效的文件路径: %s", f.Name)
	}

	// 如果是目录，创建目录
	if f.FileInfo().IsDir() {
		return os.MkdirAll(path, f.FileInfo().Mode())
	}

	// 创建父目录
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	// 打开ZIP中的文件
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	// 创建目标文件
	outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.FileInfo().Mode())
	if err != nil {
		return err
	}
	defer outFile.Close()

	// 复制数据
	_, err = io.Copy(outFile, rc)
	return err
}
