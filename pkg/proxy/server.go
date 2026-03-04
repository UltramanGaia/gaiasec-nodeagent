package proxy

import (
	"context"
	"errors"
	"gaiasec-nodeagent/pkg/pb"
	"gaiasec-nodeagent/pkg/wsclient"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
	"net"
	"nhooyr.io/websocket/wspb"
	"sync"
)

// Hub maintains the set of active proxy clients in server side for a user
type Server struct {
	WsClient *wsclient.Client
	// Registered proxy connections.
	proxyServers   map[string]*ProxyServer // 作为服务端，处理其他节点过来的代理请求
	proxyServersMu sync.RWMutex
	proxyClients   map[string]*ProxyClient // 作为客户端，处理来自本节点的代理请求
	proxyClientsMu sync.RWMutex

	sock5Addr string
	listener  net.Listener
}

func NewServer(client *wsclient.Client, address string) (*Server, error) {
	server := &Server{
		WsClient:     client,
		proxyServers: make(map[string]*ProxyServer),
		proxyClients: make(map[string]*ProxyClient),
	}
	if address != "" {
		log.Infof("Socks5 server: %s", address)
		listener, err := net.Listen("tcp", address)
		if err != nil {
			return nil, err
		}
		server.listener = listener
	}
	return server, nil
}

func (s *Server) Close() {
	// if there are connections, close them.
	s.proxyServersMu.Lock()
	defer s.proxyServersMu.Unlock()
	for id, proxy := range s.proxyServers {
		proxy.Close(false)
		delete(s.proxyServers, id)
	}
}

func (s *Server) HandleSocks5Message() {
	// 客户端连接sock5, 代理发包到对应agent
	if s.listener != nil {
		for {
			conn, err := s.listener.Accept()
			if err != nil {
				log.Error(err)
				continue
			}
			if conn == nil {
				continue
			}
			tcpConn := conn.(*net.TCPConn)
			client, err := NewClient(tcpConn, s)
			if err != nil {
				log.Errorf("Error creating client: %s", err)
				continue
			}
			go client.handleSocks5Conn()
		}
	}
}

type ClientData ServerData

var ConnCloseByClient = errors.New("conn closed by client")

func HandleProxyEstablish(server *Server, msg *pb.Base) error {
	id := msg.Session
	log.Infof("HandleProxyEstablish: session=%s, source=%s, destination=%s", id, msg.Source, msg.Destination)
	proxyEstMsg := &pb.ProxyEstablishMessage{}
	err := proto.Unmarshal(msg.Data, proxyEstMsg)
	if err != nil {
		return err
	}

	go establishProxy(server, id, proxyEstMsg.Addr, msg.Destination, msg.Source)
	return nil
}

func HandleProxyDataToServer(server *Server, msg *pb.Base) error {
	id := msg.Session
	requestMsg := &pb.ProxyData{}
	err := proto.Unmarshal(msg.Data, requestMsg)
	if err != nil {
		return err
	}

	proxy := server.GetProxyServerById(id) // 作为服务端
	if proxy != nil {
		// write income data from websocket to TCP connection
		return proxy.onData(ClientData{Tag: requestMsg.ProxyDataType, Data: requestMsg.Data})
	}
	return nil
}

func HandleProxyDataToClient(server *Server, msg *pb.Base) error {
	id := msg.Session
	requestMsg := &pb.ProxyData{}
	err := proto.Unmarshal(msg.Data, requestMsg)
	if err != nil {
		return err
	}

	// 作为客户端
	proxyClient := server.GetProxyClientById(id)
	if proxyClient != nil {
		// write income data from websocket to TCP connection
		proxyClient.onData(id, ServerData{Tag: requestMsg.ProxyDataType, Data: requestMsg.Data})
		return nil
	}

	return nil
}

func establishProxy(server *Server, sessionId string, addr string, source string, destination string) {
	log.Infof("establishProxy: session=%s, addr=%s, source=%s, destination=%s", sessionId, addr, source, destination)
	e := &ProxyServer{Id: sessionId}

	err := e.establish(server, addr, source, destination)
	if err == nil {
		log.Infof("establishProxy success: session=%s", sessionId)
		server.tellClosed(sessionId, source, destination) // tell client to close connection.
	} else if err != ConnCloseByClient {
		log.Errorf("establishProxy error: session=%s, error=%v", sessionId, err)
		server.tellClosed(sessionId, source, destination)
	}
	return
}

// data type used in DefaultProxyEstablish to pass data to channel
type ChanDone struct {
	tell bool
	err  error
}

// tell the client the connection has been closed
func (s *Server) tellClosed(id string, source string, destination string) error {
	// send finish flag to client
	base := &pb.Base{
		Type:        pb.MessageType_PROXY_CLOSE,
		Source:      source,
		Destination: destination,
		Session:     id,
	}
	// fixme lock or NextWriter
	err := wspb.Write(context.TODO(), s.WsClient.Conn, base)
	if err != nil {
		return err
	}

	return nil
}

// add a tcp connection to connection pool.
func (s *Server) addNewProxyServer(proxyInstance *ProxyServer) {
	s.proxyServersMu.Lock()
	defer s.proxyServersMu.Unlock()
	s.proxyServers[proxyInstance.Id] = proxyInstance
	log.Infof("addNewProxyServer: id=%s", proxyInstance.Id)
}

func (s *Server) GetProxyServerById(id string) *ProxyServer {
	s.proxyServersMu.RLock()
	defer s.proxyServersMu.RUnlock()
	if proxy, ok := s.proxyServers[id]; ok {
		return proxy
	}
	return nil
}

func (s *Server) removeProxyServer(id string) {
	s.proxyServersMu.Lock()
	defer s.proxyServersMu.Unlock()
	if _, ok := s.proxyServers[id]; ok {
		delete(s.proxyServers, id)
		log.Infof("removeProxyServer: id=%s", id)
	}
}

func (s *Server) addNewProxyClient(proxyInstance *ProxyClient) {
	s.proxyClientsMu.Lock()
	defer s.proxyClientsMu.Unlock()
	s.proxyClients[proxyInstance.Id] = proxyInstance
	log.Infof("addNewProxyClient: id=%s, destination=%s", proxyInstance.Id, proxyInstance.Destination)
}

func (s *Server) GetProxyClientById(id string) *ProxyClient {
	s.proxyClientsMu.RLock()
	defer s.proxyClientsMu.RUnlock()
	if proxy, ok := s.proxyClients[id]; ok {
		return proxy
	}
	return nil
}

func (s *Server) removeProxyClient(id string) {
	s.proxyClientsMu.Lock()
	defer s.proxyClientsMu.Unlock()
	if _, ok := s.proxyClients[id]; ok {
		delete(s.proxyClients, id)
		log.Infof("removeProxyClient: id=%s", id)
	}
}
