package system

import (
	"net"
	"os"
	"sort"
)

// getHostname returns the system hostname
func GetHostname() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", err
	}
	return hostname, nil
}

// GetLocalIps returns all IP address
func GetLocalIps() ([]string, error) {
	ips := []string{}
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	// 遍历每个网络接口，获取其IP地址
	for _, iface := range interfaces {
		// 过滤掉未启动的接口
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		// 过滤掉回环接口（可选）
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		// 获取接口的所有地址
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			// 解析IP地址
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil {
				continue
			}

			// 区分IPv4和IPv6
			if ip.To4() != nil {
				ips = append(ips, ip.String())
			}
		}
	}
	sort.Strings(ips)
	return ips, nil
}
