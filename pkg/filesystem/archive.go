package filesystem

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func CreateArchive(rootPath string, includes, excludes []string, archivePath string) (int64, error) {
	cleanRoot := filepath.Clean(rootPath)
	entryPrefix := archiveEntryPrefix(cleanRoot)
	rootInfo, err := os.Stat(cleanRoot)
	if err != nil {
		return 0, fmt.Errorf("stat root path: %w", err)
	}
	if !rootInfo.IsDir() {
		return 0, fmt.Errorf("root path is not a directory")
	}

	if err := os.MkdirAll(filepath.Dir(archivePath), 0o755); err != nil {
		return 0, fmt.Errorf("create archive directory: %w", err)
	}

	out, err := os.Create(archivePath)
	if err != nil {
		return 0, fmt.Errorf("create archive file: %w", err)
	}
	defer out.Close()

	writer := zip.NewWriter(out)
	defer writer.Close()

	err = filepath.WalkDir(cleanRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == cleanRoot {
			return nil
		}

		entryName := filepath.ToSlash(strings.TrimPrefix(filepath.Clean(path), string(filepath.Separator)))
		if entryPrefix != "" && strings.HasPrefix(entryName, entryPrefix+"/") {
			entryName = strings.TrimPrefix(entryName, entryPrefix+"/")
		}
		entryName = filepath.ToSlash(filepath.Clean(entryName))
		entryName = strings.TrimPrefix(entryName, "/")
		if entryName == "." || entryName == "" {
			return nil
		}
		if entryName == "" {
			return nil
		}
		if shouldExcludePath(entryName, includes, excludes) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}

		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = entryName
		header.Method = zip.Deflate

		dst, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}
		src, err := os.Open(path)
		if err != nil {
			return err
		}
		if _, err := io.Copy(dst, src); err != nil {
			src.Close()
			return err
		}
		return src.Close()
	})
	if err != nil {
		return 0, err
	}
	if err := writer.Close(); err != nil {
		return 0, err
	}
	info, err := out.Stat()
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func shouldExcludePath(entryName string, includes, excludes []string) bool {
	entryName = filepath.ToSlash(entryName)
	if len(includes) > 0 && !matchesAnyPattern(entryName, includes) {
		return true
	}
	return matchesAnyPattern(entryName, excludes)
}

func matchesAnyPattern(path string, patterns []string) bool {
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(filepath.ToSlash(pattern))
		if pattern == "" {
			continue
		}
		if matchArchivePattern(path, pattern) {
			return true
		}
	}
	return false
}

func archiveEntryPrefix(rootPath string) string {
	clean := filepath.ToSlash(filepath.Clean(rootPath))
	clean = strings.TrimPrefix(clean, "/")
	const procRootMarker = "root/"
	if strings.HasPrefix(clean, "proc/") {
		if idx := strings.Index(clean, procRootMarker); idx >= 0 {
			return clean[:idx+len(procRootMarker)-1]
		}
	}
	return ""
}

func matchArchivePattern(path, pattern string) bool {
	if ok, _ := filepath.Match(pattern, path); ok {
		return true
	}
	if strings.Contains(pattern, "**") {
		compact := strings.ReplaceAll(pattern, "**/", "")
		compact = strings.ReplaceAll(compact, "/**", "")
		compact = strings.ReplaceAll(compact, "**", "")
		if compact != "" {
			if ok, _ := filepath.Match(compact, path); ok {
				return true
			}
			if strings.HasPrefix(compact, "*/") {
				if ok, _ := filepath.Match(strings.TrimPrefix(compact, "*/"), path); ok {
					return true
				}
			}
			if strings.Contains(path, strings.Trim(compact, "*")) {
				return true
			}
		}
	}
	return false
}
