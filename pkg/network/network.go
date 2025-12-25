package network

import (
	"sothoth-nodeagent/pkg/pb"
)

// GetNetworkConnections returns a list of network connections
func GetNetworkConnections() ([]*pb.NetworkConnection, error) {
	var connections []*pb.NetworkConnection

	// Get TCP connections
	tcpSocks, err := TCPSocks(NoopFilter)
	if err != nil {
		return nil, err
	}

	for _, sock := range tcpSocks {
		pid := int32(0)
		processName := ""

		if sock.Process != nil {
			pid = int32(sock.Process.Pid)
			processName = sock.Process.Name
		}

		conn := &pb.NetworkConnection{
			Protocol:      "tcp",
			LocalAddress:  sock.LocalAddr.IP.String(),
			LocalPort:     int32(sock.LocalAddr.Port),
			RemoteAddress: sock.RemoteAddr.IP.String(),
			RemotePort:    int32(sock.RemoteAddr.Port),
			Pid:           pid,
			Uid:           0, // go-netstat库不提供UID信息
			ProcessName:   processName,
		}
		connections = append(connections, conn)
	}

	// Get UDP connections
	udpSocks, err := UDPSocks(NoopFilter)
	if err != nil {
		return nil, err
	}

	for _, sock := range udpSocks {
		pid := int32(0)
		processName := ""

		if sock.Process != nil {
			pid = int32(sock.Process.Pid)
			processName = sock.Process.Name
		}

		conn := &pb.NetworkConnection{
			Protocol:      "udp",
			LocalAddress:  sock.LocalAddr.IP.String(),
			LocalPort:     int32(sock.LocalAddr.Port),
			RemoteAddress: sock.RemoteAddr.IP.String(),
			RemotePort:    int32(sock.RemoteAddr.Port),
			Pid:           pid,
			Uid:           0, // go-netstat库不提供UID信息
			ProcessName:   processName,
		}
		connections = append(connections, conn)
	}

	return connections, nil
}
