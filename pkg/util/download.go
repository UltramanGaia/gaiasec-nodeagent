package util

import (
	"archive/zip"
	"crypto/tls"
	"fmt"
	"gaiasec-nodeagent/pkg/config"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// DownloadPlugin 下载并解压插件
func DownloadPlugin(name, version string) error {
	cfg := config.GetInstance()
	log.Infof("start download plugin: %s version: %s", name, version)
	// 构建下载URL
	downloadURL := fmt.Sprintf("/plugins/%s/%s.zip", name, version)

	// 创建插件目录
	pluginDir := filepath.Join(cfg.GaiaSecDir, "plugins", name, version)
	err := MkdirAll(pluginDir, 0755)
	if err != nil {
		return fmt.Errorf("create plugin dir error: %v", err)
	}

	// 下载插件文件
	zipPath := filepath.Join(pluginDir, "..", fmt.Sprintf("%s-%s.zip", name, version))
	err = downloadFile(downloadURL, zipPath)
	if err != nil {
		return fmt.Errorf("download plugin error: %v", err)
	}

	// 解压插件文件
	err = extractZip(zipPath, pluginDir)
	if err != nil {
		return fmt.Errorf("unzip plugin error: %v", err)
	}

	// 删除Zip文件
	os.Remove(zipPath)

	log.Infof("plugin %s download and unzip success", name)
	return nil
}

// DownloadTool 下载工具
func DownloadTool(name string) error {
	cfg := config.GetInstance()
	log.Infof("start download tool: %s", name)
	// 构建下载URL
	downloadURL := fmt.Sprintf("/plugins/nodeagent/%s-%s-%s", name, runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		downloadURL += ".exe"
		name += ".exe"
	}

	// 下载插件文件
	toolPath := filepath.Join(cfg.GaiaSecDir, name)
	err := downloadFile(downloadURL, toolPath)
	if err != nil {
		return fmt.Errorf("download tool error: %v", err)
	}

	os.Chmod(toolPath, 0755)

	log.Infof("tool %s download success", name)
	return nil
}

// downloadFile 下载文件
func downloadFile(url, filepath string) error {
	url = config.GetInstance().Server + url

	log.Infof("download file: %s", url)

	// 创建自定义HTTP客户端，禁用TLS证书验证
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	// 创建HTTP请求
	resp, err := httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP response error: %d %s", resp.StatusCode, resp.Status)
	}

	// 创建目标文件
	out, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		return fmt.Errorf("create file error: %v", err)
	}
	defer out.Close()

	// 复制数据
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("write file error: %v", err)
	}

	_ = EnsureFilePerm(filepath)
	log.Infof("download file success: %s", filepath)
	return nil
}

// extractZip 解压ZIP文件
func extractZip(src, dest string) error {
	log.Infof("unzip file: %s to %s", src, dest)

	// 打开ZIP文件
	r, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("open zip file error: %v", err)
	}
	defer r.Close()

	// 创建目标目录
	err = os.MkdirAll(dest, 0777)
	if err != nil {
		return fmt.Errorf("create target dir error: %v", err)
	}

	// 解压文件
	for _, f := range r.File {
		err := extractFile(f, dest)
		if err != nil {
			return fmt.Errorf("unzip file %s error: %v", f.Name, err)
		}
	}

	log.Infof("unzip file success")
	return nil
}

// extractFile 解压单个文件
func extractFile(f *zip.File, destDir string) error {
	// 构建目标路径
	path := filepath.Join(destDir, f.Name)

	// 检查路径安全性（防止目录遍历攻击）
	if !strings.HasPrefix(path, filepath.Clean(destDir)+string(os.PathSeparator)) {
		return fmt.Errorf("error path: %s", f.Name)
	}

	// 如果是目录，创建目录
	if f.FileInfo().IsDir() {
		return MkdirAll(path, 0777)
	}

	// 创建父目录
	if err := MkdirAll(filepath.Dir(path), 0777); err != nil {
		return err
	}

	// 打开ZIP中的文件
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	// 创建目标文件
	outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// 复制数据
	_, err = io.Copy(outFile, rc)

	// set permission
	_ = EnsureFilePerm(path)

	return err
}
