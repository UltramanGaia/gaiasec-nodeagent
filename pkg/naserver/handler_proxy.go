package naserver

import (
	"sothoth-nodeagent/pkg/pb"
	"sothoth-nodeagent/pkg/proxy"
)

// 代理相关处理方法
func (na *NodeAgent) handleProxyClose(message *pb.Base) {
	id := message.Session
	proxyInstance := na.proxyServer.GetProxyServerById(id)
	if proxyInstance != nil {
		proxyInstance.ProxyIns.Close(false) // todo remove proxy here
	}
}

func (na *NodeAgent) handleProxyEstablish(message *pb.Base) {
	proxy.HandleProxyEstablish(na.proxyServer, message)
}

func (na *NodeAgent) handleProxyData(message *pb.Base) {
	proxy.HandleProxyData(na.proxyServer, message)
}
