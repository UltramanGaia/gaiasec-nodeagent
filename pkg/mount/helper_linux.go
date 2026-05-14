//go:build linux

package mount

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	"gaiasec-nodeagent/pkg/util"
)

func runMounter(args ...string) error {
	mounter, err := util.Tool("mounter")
	if err != nil {
		return fmt.Errorf("cannot found mounter: %v", err)
	}

	cmd := exec.Command(mounter, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			return fmt.Errorf("mounter execute failed: %v", err)
		}
		return fmt.Errorf("mounter execute failed: %v, Output: %s", err, message)
	}

	if len(output) > 0 {
		log.Infof("mounter output: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

func DoMount(src string, dest string, pid int) error {
	return runMounter(
		"ensure-path-visible",
		"--target-pid", fmt.Sprintf("%d", pid),
		"--src", src,
		"--dest", dest,
	)
}

func TryMountDir(pid int, src string, dest string) error {
	log.Infof("try mount dir %s to %s", src, dest)

	targetPath := filepath.Join("/proc", strconv.Itoa(pid), "root", strings.TrimPrefix(dest, "/"))
	if _, err := os.Stat(targetPath); err == nil {
		log.Infof("skip mount dir %s to %s because target already exists: %s", src, dest, targetPath)
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat mount target %s failed: %w", targetPath, err)
	}

	return DoMount(src, dest, pid)
}

func RemountRoot(pid int) error {
	return runMounter(
		"remount-root-if-ro",
		"--target-pid", fmt.Sprintf("%d", pid),
	)
}
