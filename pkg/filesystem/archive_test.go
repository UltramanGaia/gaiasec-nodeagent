package filesystem

import (
	"archive/zip"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCreateArchiveSkipsDirectorySymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory symlink creation differs on windows")
	}

	root := t.TempDir()
	realDir := filepath.Join(root, "config", "neteco")
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatalf("mkdir real dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "app.yml"), []byte("ok"), 0o644); err != nil {
		t.Fatalf("write regular file: %v", err)
	}
	if err := os.Symlink(realDir, filepath.Join(root, "neteco-link")); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	archivePath := filepath.Join(t.TempDir(), "bundle.zip")
	if _, err := CreateArchive(root, nil, nil, archivePath); err != nil {
		t.Fatalf("CreateArchive() error = %v", err)
	}

	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatalf("open archive: %v", err)
	}
	defer reader.Close()

	entries := make(map[string]struct{}, len(reader.File))
	for _, file := range reader.File {
		entries[file.Name] = struct{}{}
	}

	foundRegularFile := false
	for name := range entries {
		if strings.HasSuffix(name, "/app.yml") || name == "app.yml" {
			foundRegularFile = true
			break
		}
	}
	if !foundRegularFile {
		t.Fatalf("archive missing regular file")
	}
	for name := range entries {
		if strings.HasSuffix(name, "/neteco-link") || name == "neteco-link" {
			t.Fatalf("archive should skip directory symlink")
		}
	}
}
