package filesystem

import (
	"encoding/base64"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// FileContent represents file content with metadata
type FileContent struct {
	Content      string `json:"content"`
	Encoding     string `json:"encoding"`
	Size         int64  `json:"size"`
	LastModified int64  `json:"lastModified"`
	IsBinary     bool   `json:"isBinary"`
}

// FileNode represents a file or directory node
type FileNode struct {
	Name         string      `json:"name"`
	Path         string      `json:"path"`
	Type         string      `json:"type"` // "file" or "directory"
	Size         int64       `json:"size"`
	LastModified int64       `json:"lastModified"`
	Permissions  string      `json:"permissions"`
	Link         string      `json:"link"`
	IsHidden     bool        `json:"isHidden"`
	Children     []*FileNode `json:"children,omitempty"`
}

// ListDirectory lists files and directories in the given path
func ListDirectory(path string) ([]*FileNode, error) {
	cleanPath := filepath.Clean(path)
	entries, err := os.ReadDir(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var nodes []*FileNode
	for _, entry := range entries {
		node, err := createFileNode(cleanPath, entry)
		if err != nil {
			// Log error but continue with other files
			continue
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}

// ReadFile reads a file and returns its content with metadata
func ReadFile(path string) (*FileContent, error) {
	// Get file info first
	node, err := GetFileInfo(path)
	if err != nil {
		return nil, err
	}

	if node.Type == "directory" {
		return nil, fmt.Errorf("cannot read directory as file")
	}

	// Check if file is binary
	isBinary := IsBinaryFile(path)

	content := &FileContent{
		Size:         node.Size,
		LastModified: node.LastModified,
		IsBinary:     isBinary,
	}

	if isBinary {
		// For binary files, read as base64
		data, err := readFileContent(path)
		if err != nil {
			return nil, err
		}
		content.Content = base64.StdEncoding.EncodeToString(data)
		content.Encoding = "base64"
	} else {
		// For text files, read as UTF-8
		data, err := readFileContent(path)
		if err != nil {
			return nil, err
		}
		content.Content = string(data)
		content.Encoding = "utf-8"
	}

	return content, nil
}

// WriteFile writes content to a file
func WriteFile(path, content string) error {
	// Detect if content is base64 encoded
	var data []byte
	var err error

	if isBase64(content) {
		data, err = base64.StdEncoding.DecodeString(content)
		if err != nil {
			return fmt.Errorf("failed to decode base64 content: %w", err)
		}
	} else {
		data = []byte(content)
	}

	return writeFileContent(path, data)
}

// CreateFile creates a new empty file
func CreateFile(path string) error {
	return writeFileContent(path, []byte{})
}

// CreateDirectory creates a new directory
func CreateDirectory(path string) error {
	cleanPath := filepath.Clean(path)
	err := os.MkdirAll(cleanPath, 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	return nil
}

// Delete removes a file or directory
func Delete(path string) error {
	cleanPath := filepath.Clean(path)
	err := os.RemoveAll(cleanPath)
	if err != nil {
		return fmt.Errorf("failed to delete: %w", err)
	}
	return nil
}

// Rename renames/moves a file or directory
func Rename(oldPath, newPath string) error {
	cleanOldPath := filepath.Clean(oldPath)
	cleanNewPath := filepath.Clean(newPath)
	err := os.Rename(cleanOldPath, cleanNewPath)
	if err != nil {
		return fmt.Errorf("failed to rename: %w", err)
	}
	return nil
}

// GetFileInfo gets information about a specific file or directory
func GetFileInfo(path string) (*FileNode, error) {
	cleanPath := filepath.Clean(path)
	info, err := os.Stat(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	return createFileNodeFromInfo(cleanPath, info)
}

// Helper functions

// readFileContent reads the content of a file
func readFileContent(path string) ([]byte, error) {
	cleanPath := filepath.Clean(path)
	// Check if it's a file
	info, err := os.Stat(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	if info.IsDir() {
		return nil, fmt.Errorf("cannot read directory as file")
	}

	// Check file size to prevent reading very large files
	if info.Size() > 10*1024*1024 { // 10MB limit
		return nil, fmt.Errorf("file too large to read (max 10MB)")
	}

	content, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return content, nil
}

// writeFileContent writes content to a file
func writeFileContent(path string, content []byte) error {
	cleanPath := filepath.Clean(path)
	// Create directory if it doesn't exist
	dir := filepath.Dir(cleanPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	err := os.WriteFile(cleanPath, content, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func createFileNode(parentPath string, entry fs.DirEntry) (*FileNode, error) {
	info, err := entry.Info()
	if err != nil {
		return nil, err
	}

	fullPath := filepath.Join(parentPath, entry.Name())
	return createFileNodeFromInfo(fullPath, info)
}

func createFileNodeFromInfo(path string, info fs.FileInfo) (*FileNode, error) {
	nodeType := "file"
	if info.IsDir() {
		nodeType = "directory"
	}
	mode := info.Mode()
	link := ""
	if mode&fs.ModeSymlink == fs.ModeSymlink {
		nodeType = "symlink"
		link, _ = os.Readlink(path)
	}

	return &FileNode{
		Name:         info.Name(),
		Path:         path,
		Type:         nodeType,
		Size:         info.Size(),
		LastModified: info.ModTime().UnixMilli(),
		Permissions:  info.Mode().String(),
		Link:         link,
		IsHidden:     strings.HasPrefix(info.Name(), "."),
	}, nil
}

// IsBinaryFile checks if a file is binary by examining its content
func IsBinaryFile(path string) bool {
	cleanPath := filepath.Clean(path)
	file, err := os.Open(cleanPath)
	if err != nil {
		return false
	}
	defer file.Close()

	// Read first 512 bytes to detect binary content
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && n == 0 {
		return false
	}

	// Check for null bytes which typically indicate binary content
	for i := 0; i < n; i++ {
		if buffer[i] == 0 {
			return true
		}
	}

	return false
}

// isBase64 detects if string is base64 encoded
func isBase64(s string) bool {
	// Simple heuristic: if string contains only base64 characters and has proper padding
	if len(s)%4 != 0 {
		return false
	}

	for _, c := range s {
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
			(c >= '0' && c <= '9') || c == '+' || c == '/' || c == '=') {
			return false
		}
	}

	// Additional check: base64 strings typically don't contain common text patterns
	return !strings.Contains(strings.ToLower(s), "the ") &&
		!strings.Contains(strings.ToLower(s), "and ") &&
		len(s) > 20 // Minimum length for base64 detection
}
