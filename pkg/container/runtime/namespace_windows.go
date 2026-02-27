//go:build windows

package runtime

// GetPIDNamespace 返回给定进程的 PID 命名空间 inode 字符串。
// Windows 不支持 PID 命名空间，返回空字符串。
func GetPIDNamespace(pid int) (string, error) {
	return "", nil
}
