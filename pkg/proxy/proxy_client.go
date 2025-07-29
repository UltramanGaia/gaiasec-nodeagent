package proxy

import "sothoth-nodeagent/pkg/pb"

// proxy client handle one connection, send data to proxy server vai websocket.
type ProxyClient struct {
	Id          string
	Source      string
	Destination string
	onData      func(string, ServerData) // data from server todo data with  type
	onClosed    func(string, bool)       // close connection, param bool: do tellClose if true
	onError     func(string, error)      // if there are error messages
}

// tell wssocks proxy server to establish a proxy connection by sending server
// proxy address, type, initial data.
func (p *ProxyClient) Establish(client *Client, addr string) error {
	m := &pb.ProxyEstablishMessage{
		Addr: addr,
	}
	return client.server.WsClient.SendMessage(m, pb.MessageType_PROXY_ESTABLISH, p.Source, p.Destination, p.Id)
}
