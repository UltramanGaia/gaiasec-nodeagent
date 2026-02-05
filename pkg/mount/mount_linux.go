//go:build linux && cgo

package mount

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"gaiasec-nodeagent/pkg/util"
	"strings"
)

/*
#cgo CFLAGS: -Wall
#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/wait.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/mount.h>
#include <sched.h>
#include <fcntl.h>
#include <string.h>
#include <sys/sysmacros.h>

void cmount(const char *nsPath, int majorId, int minorId, const char *fsType, const char *sourcePath, const char *destPath) {
	pid_t pid = fork();
    int status;
    if(pid == 0){
	    const char * sdaPath = "/tmpsda";
	    const char * hostFs = "/tmpmnt";
	    int fd = open(nsPath, O_RDONLY);
	    int res;
		setns(fd, 0);
		// 1. mknod --mode 0600 /dev/block-mount $DEV_major $DEV_minor
		 mknod(sdaPath, S_IFBLK | 0600, makedev(majorId, minorId));
		// 2. mkdir /tmpmnt
		mkdir(hostFs, 0777);
		// 3. mount /dev/block-mount /tmpmnt
		res = mount(sdaPath, hostFs, fsType, MS_MGC_VAL, NULL);
		if (res != 0){
			perror("mount hostfs 1");
			exit(-1);
		}
		// 4. mount -o bind /tmpmnt/runtime /runtime
		char src[128];
		sprintf(src, "%s%s",hostFs, sourcePath);
		res = mount(src, destPath, NULL, MS_BIND|MS_SILENT, NULL);
		if (res != 0){
			perror("mount hostfs 2");
			exit(-1);
		}
		// 5. umount /tmpmnt
		umount(hostFs);
		// 6. rm -rf /tmpmnt
		rmdir(hostFs);
		exit(0);
	}else{
		wait(&status);
	}
}

void cremount(const char *nsPath, const char *target) {
	pid_t pid = fork();
	int status;
	if(pid == 0){
		int fd = open(nsPath, O_RDONLY);
		int res;
		setns(fd, 0);

		// mount -o remount,rw /
		res = mount(NULL, target, NULL, MS_REMOUNT|MS_RELATIME, NULL);
		if(res!=0){
			perror("remount / error");
			exit(-1);
		}
		exit(0);
	}else{
		wait(&status);
	}
}


*/
import "C"

func DoMount(src string, dest string, pid int) error {
	info, err := os.Stat(src)
	if err != nil {
		if !os.IsExist(err) {
			err = fmt.Errorf("source directory not found: %s", src)
			return err
		}
		return err
	}
	srcMod := info.Mode()
	srcIsDir := srcMod.IsDir()
	destPath := fmt.Sprintf("/proc/%d/root%s", pid, dest)
	if !util.Exists(destPath) {
		if srcIsDir {
			util.MkdirAll(destPath, srcMod)
		} else {
			destParentPaht := filepath.Dir(destPath)
			if !util.Exists(destParentPaht) {
				util.MkdirAll(destParentPaht, os.ModePerm)
			}
			os.OpenFile(destPath, os.O_CREATE, srcMod)
		}
	}
	return realMount(src, dest, pid)
}

func realMount(src string, dest string, pid int) error {
	ret, err := GetPartitions("/proc/self/mountinfo")
	if err != nil {
		return err
	}
	if len(ret) == 0 {
		return fmt.Errorf("no mount info found")
	}
	var match bool
	var matchItem PartitionStat
	for _, part := range ret {
		if strings.HasPrefix(src, part.Mountpoint) {
			if match == false || len(part.Mountpoint) > len(matchItem.Mountpoint) {
				matchItem = part
				match = true
			}
		}
	}

	if !match {
		return fmt.Errorf("no mount point found for %s", src)
	}

	mntPath := fmt.Sprintf("/proc/%d/ns/mnt", pid)
	nsPath := C.CString(mntPath)
	majorId := C.int(matchItem.Major)
	minorId := C.int(matchItem.Minor)
	fsType := C.CString(matchItem.Fstype)
	sourcePath := C.CString(src)
	destPath := C.CString(dest)
	log.Infof("try mount with c %s, %d, %d, %s, %s, %s", mntPath, matchItem.Major, matchItem.Minor, matchItem.Fstype, src, dest)
	C.cmount(nsPath, majorId, minorId, fsType, sourcePath, destPath)
	return nil
}

func TryMountDir(pid int, src string, dest string) error {
	log.Infof("try mount dir %s to %s", src, dest)
	rootPrefix := fmt.Sprintf("/proc/%d/root", pid)
	if util.Exists(rootPrefix + "/") {
		destDir := rootPrefix + dest
		if !util.Exists(destDir) {
			err := tryRemountRootPath(pid)
			if err != nil {
				log.Errorf("try remount root path error: %s", err)
				return err
			}
			err = util.MkdirAll(destDir, os.ModePerm)
			if err != nil {
				log.Errorf("mkdir error: %s", err)
				return err
			}
			err = DoMount(src, dest, pid)
			if err != nil {
				log.Errorf("mount error: %s", err)
			}
			return err
		} else {
			log.Warnf("dest dir exists: %s", destDir)
		}
	} else {
		log.Warnf("root path not exists: %s", rootPrefix)
	}
	return nil
}

func RemountRoot(pid int) error {
	mntPath := fmt.Sprintf("/proc/%d/ns/mnt", pid)
	nsPath := C.CString(mntPath)
	target := C.CString("/")
	log.Infof("try remount root path with c %s, %s", mntPath, "/")
	C.cremount(nsPath, target)
	return nil
}

func tryRemountRootPath(pid int) error {
	mountInfo := fmt.Sprintf("/proc/%d/mountinfo", pid)
	parts, err := GetPartitions(mountInfo)
	if err != nil {
		return err
	}
	for _, part := range parts {
		if part.Mountpoint == "/" {
			if strings.HasPrefix(part.Options, "ro,") {
				// need remount ro
				return RemountRoot(pid)
			}
		}
	}
	return nil
}
