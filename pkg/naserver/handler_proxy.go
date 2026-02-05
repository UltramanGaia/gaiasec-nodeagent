package naserver

import (
	"gaiasec-nodeagent/pkg/pb"
	"gaiasec-nodeagent/pkg/proxy"
)

// 代理相关处理方法
func (na *NodeAgent) handleProxyClose(message *pb.Base) {
	id := message.Session
	proxyServer := na.proxyServer.GetProxyServerById(id)
	if proxyServer != nil {
		proxyServer.Close(false) // todo remove proxy here
	}
	proxyClient := na.proxyServer.GetProxyClientById(id)
	if proxyClient != nil {
		proxyClient.Close(false) // todo remove proxy here
	}
}

func (na *NodeAgent) handleProxyEstablish(message *pb.Base) {
	proxy.HandleProxyEstablish(na.proxyServer, message)
}

func (na *NodeAgent) handleProxyDataToServer(message *pb.Base) {
	proxy.HandleProxyDataToServer(na.proxyServer, message)
}

func (na *NodeAgent) handleProxyDataToClient(message *pb.Base) {
	proxy.HandleProxyDataToClient(na.proxyServer, message)
}
