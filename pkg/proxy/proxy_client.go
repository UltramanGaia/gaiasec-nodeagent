package proxy

import "gaiasec-nodeagent/pkg/pb"

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
func (pc *ProxyClient) Establish(client *Client, addr string) error {
	m := &pb.ProxyEstablishMessage{
		Addr: addr,
	}
	return client.server.WsClient.SendMessage(m, pb.MessageType_PROXY_ESTABLISH, pc.Source, pc.Destination, pc.Id)
}

func (pc *ProxyClient) Close(tell bool) error {
	pc.onClosed(pc.Id, tell)
	return nil // todo error
}
