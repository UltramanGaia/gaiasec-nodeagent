package util

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// MkdirAllWithPerm creates a directory with specified permissions
func MkdirAllWithPerm(path string, perm os.FileMode) error {
	dir, err := os.Stat(path)
	if err == nil {
		if dir.IsDir() {
			return nil
		}
		return fmt.Errorf("path %s is not a directory", path)
	}

	parentDir := filepath.Dir(path)
	if err := MkdirAllWithPerm(parentDir, perm); err != nil {
		return fmt.Errorf("failed to create parent directory %s: %v", parentDir, err)
	}

	// Create directory if it doesn't exist
	if err := MkdirWithPerm(path, perm); err != nil {
		return fmt.Errorf("failed to create directory %s: %v", path, err)
	}

	return nil
}

// MkdirWithPerm creates a directory with specified permissions
func MkdirWithPerm(path string, perm os.FileMode) error {
	// Create directory if it doesn't exist
	if err := os.Mkdir(path, perm); err != nil {
		return fmt.Errorf("failed to create directory %s: %v", path, err)
	}

	// Ensure the directory has the correct permissions
	if err := os.Chmod(path, perm); err != nil {
		return fmt.Errorf("failed to set permissions for directory %s: %v", path, err)
	}

	return nil
}

// EnsureFileDir ensures the parent directory of a file exists
func EnsureFileDir(filePath string) error {
	dir := filepath.Dir(filePath)
	return MkdirAllWithPerm(dir, 0777)
}

func Exists(filePath string) bool {
	_, err := os.Stat(filePath)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

func IsDirEmpty(filePath string) bool {
	f, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer f.Close()

	entries, _ := f.ReadDir(1)
	return len(entries) == 0
}

// ReadLines 从文件中读取数据并按行分割
// 是ReadLinesOffsetN(filename, 0, -1)的封装方法
func ReadLines(filename string) ([]string, error) {
	return ReadLinesOffsetN(filename, 0, -1)
}

// ReadLinesOffsetN 从文件中读取数据并以`\n`分割
// `offset`是指应该从哪一行开始读
// `n` 是指从offset行开始读多少行(包含`offset`行)
// n >= 0: 读取n行
// n < 0 整个文件
func ReadLinesOffsetN(filename string, offset, n int) ([]string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return []string{""}, err
	}
	defer f.Close()

	var ret []string
	r := bufio.NewReader(f)
	for i := 0; i < n+int(offset) || n < 0; i++ {
		line, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF && len(line) > 0 {
				ret = append(ret, strings.Trim(line, "\n"))
			}
			break
		}
		if i < int(offset) {
			continue
		}
		ret = append(ret, strings.Trim(line, "\n"))
	}

	return ret, nil
}
