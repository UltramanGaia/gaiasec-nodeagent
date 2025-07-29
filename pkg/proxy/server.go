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
	ConnPool  map[string]*ProxyServer
	sock5Addr string
	listener  net.Listener
	mu        sync.RWMutex
}

type ProxyServer struct {
	Id       string // id of proxy connection
	ProxyIns *DefaultProxyEst
}

type ProxyRegister struct {
	id       string
	addr     string
	withData []byte
}

func NewServer(client *wsclient.Client, address string) (*Server, error) {
	server := &Server{
		WsClient: client,
		ConnPool: make(map[string]*ProxyServer),
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

func (s *Server) GetProxyById(id string) *ProxyServer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if proxy, ok := s.ConnPool[id]; ok {
		return proxy
	}
	return nil
}

type Connector struct {
	Conn io.ReadWriteCloser
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

	var estData []byte = nil
	if proxyEstMsg.WithData {
		estData = proxyEstMsg.Data
	}
	go establishProxy(server, ProxyRegister{id, proxyEstMsg.Addr, estData}, msg.Destination, msg.Source)
	return nil
}

func HandleProxyData(server *Server, msg *pb.Base) error {
	id := msg.Session
	requestMsg := &pb.ProxyData{}
	err := proto.Unmarshal(msg.Data, requestMsg)
	if err != nil {
		return err
	}

	if proxy := server.GetProxyById(id); proxy != nil {
		// write income data from websocket to TCP connection
		return proxy.ProxyIns.onData(ClientData{Tag: requestMsg.ProxyDataType, Data: requestMsg.Data})
	}
	return nil
}

func establishProxy(server *Server, proxyMeta ProxyRegister, source string, destination string) {
	e := &DefaultProxyEst{}

	err := e.establish(server, proxyMeta.id, proxyMeta.addr, source, destination)
	if err == nil {
		server.tellClosed(proxyMeta.id, source, destination) // tell client to close connection.
	} else if err != ConnCloseByClient {
		log.Error(err) // todo error handle better way
		server.tellClosed(proxyMeta.id, source, destination)
	}
	return
	//	log.WithField("size", s.GetConnectorSize()).Trace("connection size changed.")
}

// data type used in DefaultProxyEst to pass data to channel
type ChanDone struct {
	tell bool
	err  error
}

// interface implementation for socks5 and https proxy.
type DefaultProxyEst struct {
	done    chan ChanDone
	tcpConn net.Conn
}

func (e *DefaultProxyEst) onData(data ClientData) error {
	if _, err := e.tcpConn.Write(data.Data); err != nil {
		e.done <- ChanDone{true, err}
	}
	return nil
}

func (e *DefaultProxyEst) Close(tell bool) error {
	e.done <- ChanDone{tell, ConnCloseByClient}
	return nil // todo error
}

// data: data send in establish step (can be nil).
func (e *DefaultProxyEst) establish(s *Server, id string, addr string, source string, destination string) error {
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := s.WsClient.WriteProxyMessage(ctx, id, pb.PROXY_DATA_TYPE_DATA, []byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, source, destination); err != nil {
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
