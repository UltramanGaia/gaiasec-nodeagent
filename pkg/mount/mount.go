//go:build !linux || !cgo

package mount

import log "github.com/sirupsen/logrus"

func DoMount(src string, dest string, pid int) error {
	return nil
}

func TryMountDir(pid int, src string, dest string) error {
	log.Infof("try mount dir %s to %s but do nothing", src, dest)
	return nil
}

func RemountRoot(pid int) error {
	return nil
}
