package proxy

import (
	"context"
	"errors"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"nhooyr.io/websocket/wspb"
	"sothoth-nodeagent/pkg/pb"
	"sothoth-nodeagent/pkg/wsclient"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"
)

// Hub maintains the set of active proxy clients in server side for a user
type Server struct {
	WsClient *wsclient.Client
	// Registered proxy connections.
	ConnPool map[string]*ProxyServer

	proxies   map[string]*ProxyClient // all proxies on this websocket.
	sock5Addr string
	listener  net.Listener
	mu        sync.RWMutex
}

func NewServer(client *wsclient.Client, address string) (*Server, error) {
	server := &Server{
		WsClient: client,
		ConnPool: make(map[string]*ProxyServer),
		proxies:  make(map[string]*ProxyClient),
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
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, proxy := range s.ConnPool {
		proxy.ProxyIns.Close(false)
		delete(s.ConnPool, id)
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
			go client.HandleSocks5Conn()
		}
	}
}

func (s *Server) GetProxyServerById(id string) *ProxyServer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if proxy, ok := s.ConnPool[id]; ok {
		return proxy
	}
	return nil
}

func (s *Server) GetProxyClientById(id string) *ProxyClient {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if proxy, ok := s.proxies[id]; ok {
		return proxy
	}
	return nil
}

type ClientData ServerData

var ConnCloseByClient = errors.New("conn closed by client")

func HandleProxyEstablish(server *Server, msg *pb.Base) error {
	id := msg.Session
	proxyEstMsg := &pb.ProxyEstablishMessage{}
	err := proto.Unmarshal(msg.Data, proxyEstMsg)
	if err != nil {
		return err
	}

	go establishProxy(server, id, proxyEstMsg.Addr, msg.Destination, msg.Source)
	return nil
}

func HandleProxyData(server *Server, msg *pb.Base) error {
	id := msg.Session
	requestMsg := &pb.ProxyData{}
	err := proto.Unmarshal(msg.Data, requestMsg)
	if err != nil {
		return err
	}

	proxy := server.GetProxyServerById(id) // 作为服务端
	if proxy != nil {
		// write income data from websocket to TCP connection
		return proxy.ProxyIns.onData(ClientData{Tag: requestMsg.ProxyDataType, Data: requestMsg.Data})
	} else {
		// 作为客户端
		proxyClient := server.GetProxyClientById(id)
		if proxyClient != nil {
			// write income data from websocket to TCP connection
			proxyClient.onData(id, ServerData{Tag: requestMsg.ProxyDataType, Data: requestMsg.Data})
			return nil
		}
	}
	return nil
}

func establishProxy(server *Server, sessionId string, addr string, source string, destination string) {
	e := &DefaultProxyEstablish{}

	err := e.establish(server, sessionId, addr, source, destination)
	if err == nil {
		server.tellClosed(sessionId, source, destination) // tell client to close connection.
	} else if err != ConnCloseByClient {
		log.Error(err) // todo error handle better way
		server.tellClosed(sessionId, source, destination)
	}
	return
}

// data type used in DefaultProxyEstablish to pass data to channel
type ChanDone struct {
	tell bool
	err  error
}

// interface implementation for socks5 proxy.
type DefaultProxyEstablish struct {
	done    chan ChanDone
	tcpConn net.Conn
}

func (e *DefaultProxyEstablish) onData(data ClientData) error {
	if _, err := e.tcpConn.Write(data.Data); err != nil {
		e.done <- ChanDone{true, err}
	}
	return nil
}

func (e *DefaultProxyEstablish) Close(tell bool) error {
	e.done <- ChanDone{tell, ConnCloseByClient}
	return nil // todo error
}

// data: data send in establish step (can be nil).
func (e *DefaultProxyEstablish) establish(s *Server, id string, addr string, source string, destination string) error {
	conn, err := net.DialTimeout("tcp", addr, time.Second*8) // todo config timeout
	if err != nil {
		return err
	}
	e.tcpConn = conn
	defer conn.Close()

	e.done = make(chan ChanDone, 2)
	//defer close(done)

	// todo check exists
	s.addNewProxy(&ProxyServer{Id: id, ProxyIns: e})
	defer s.RemoveProxy(id)

	bytes := []byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	m := &pb.ProxyData{
		ProxyDataType: pb.PROXY_DATA_TYPE_DATA,
		Data:          bytes,
	}

	err = s.WsClient.SendMessage(m, pb.MessageType_PROXY_DATA, source, destination, id)
	if err != nil {
		return err
	}

	go func() {
		writer := NewWebSocketWriter(s.WsClient, id, context.Background(), source, destination)
		if _, err := io.Copy(writer, conn); err != nil {
			log.Error("copy error,", err)
			e.done <- ChanDone{true, err}
		}
		e.done <- ChanDone{true, nil}
	}()

	d := <-e.done
	// s.RemoveProxy(proxy.Id)
	// tellClosed is called outside this func.
	return d.err
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
func (s *Server) addNewProxy(proxyInstance *ProxyServer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ConnPool[proxyInstance.Id] = proxyInstance
}

func (s *Server) RemoveProxy(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.ConnPool[id]; ok {
		delete(s.ConnPool, id)
	}
}
