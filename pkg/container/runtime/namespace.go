package runtime

import (
	"strconv"
	"syscall"
)

// GetPIDNamespace 返回给定进程的 PID 命名空间 inode 字符串。
// 为空或错误时返回空字符串和相应错误。
func GetPIDNamespace(pid int) (string, error) {
	if pid <= 0 {
		return "", nil
	}

	path := "/proc/" + strconv.Itoa(pid) + "/ns/pid"

	var stat syscall.Stat_t
	err := syscall.Stat(path, &stat)
	if err != nil {
		return "", err
	}

	return strconv.FormatUint(uint64(stat.Ino), 10), nil
}
