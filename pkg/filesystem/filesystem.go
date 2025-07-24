package filesystem

import (
	"encoding/base64"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sothoth-nodeagent/pkg/pb"
	"strings"
)

// ListDirectory lists files and directories in the given path
func ListDirectory(path string) ([]*pb.FileNode, error) {
	cleanPath := filepath.Clean(path)
	entries, err := os.ReadDir(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var nodes []*pb.FileNode
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
func ReadFile(path string) (*pb.FSReadFileResponse, error) {
	// Get file info first
	node, err := GetFileInfo(path)
	if err != nil {
		return nil, err
	}

	if node.FileType == pb.FileType_DIRECTORY {
		return nil, fmt.Errorf("cannot read directory as file")
	}

	isBinary := IsBinaryFile(path)

	content := ""
	if isBinary {
		// For binary files, read as base64
		data, err := readFileContent(path)
		if err != nil {
			return nil, err
		}
		content = base64.StdEncoding.EncodeToString(data)
	} else {
		// For text files, read as UTF-8
		data, err := readFileContent(path)
		if err != nil {
			return nil, err
		}
		content = string(data)
	}

	response := &pb.FSReadFileResponse{
		Path:     path,
		Content:  content,
		Size:     node.Size,
		IsBinary: isBinary,
		Result:   "ok",
	}
	return response, nil
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
func GetFileInfo(path string) (*pb.FileNode, error) {
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

func createFileNode(parentPath string, entry fs.DirEntry) (*pb.FileNode, error) {
	info, err := entry.Info()
	if err != nil {
		return nil, err
	}

	fullPath := filepath.Join(parentPath, entry.Name())
	return createFileNodeFromInfo(fullPath, info)
}

func createFileNodeFromInfo(path string, info fs.FileInfo) (*pb.FileNode, error) {
	fileType := pb.FileType_FILE
	if info.IsDir() {
		fileType = pb.FileType_DIRECTORY
	}
	mode := info.Mode()
	link := ""
	if mode&fs.ModeSymlink == fs.ModeSymlink {
		fileType = pb.FileType_SYMLINK
		link, _ = os.Readlink(path)
	}

	return &pb.FileNode{
		Name:         info.Name(),
		Path:         path,
		FileType:     fileType,
		Size:         info.Size(),
		ModifiedTime: info.ModTime().UnixMilli(),
		Permission:   info.Mode().String(),
		Link:         link,
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
