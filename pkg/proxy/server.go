package proxy

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"net/http"
	"nhooyr.io/websocket/wspb"
	"sothoth-nodeagent/pkg/pb"
	"sothoth-nodeagent/pkg/util"
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
	ProxyIns ProxyEstablish
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

// interface of establishing proxy connection with target
type ProxyEstablish interface {
	establish(server *Server, id string, addr string, data []byte) error

	// data from client todo data with type
	onData(data ClientData) error

	// close connection
	// tell: whether to send close message to proxy client
	Close(tell bool) error
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
	go establishProxy(server, ProxyRegister{id, proxyEstMsg.Addr, estData})
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

func establishProxy(server *Server, proxyMeta ProxyRegister) {
	var e ProxyEstablish
	e = makeHttpProxyInstance()

	err := e.establish(server, proxyMeta.id, proxyMeta.addr, proxyMeta.withData)
	if err == nil {
		server.tellClosed(proxyMeta.id) // tell client to close connection.
	} else if err != ConnCloseByClient {
		log.Error(err) // todo error handle better way
		server.tellClosed(proxyMeta.id)
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
func (e *DefaultProxyEst) establish(s *Server, id string, addr string, data []byte) error {
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
	if err := s.WsClient.WriteProxyMessage(ctx, id, pb.PROXY_DATA_TYPE_DATA, []byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}); err != nil {
		return err
	}

	go func() {
		writer := NewWebSocketWriter(s.WsClient, id, context.Background())
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

type HttpProxyEst struct {
	bodyReadCloser *BufferedWR
}

func makeHttpProxyInstance() *HttpProxyEst {
	buf := NewBufferWR()
	return &HttpProxyEst{bodyReadCloser: buf}
}

func (h *HttpProxyEst) onData(data ClientData) error {
	if data.Tag == pb.PROXY_DATA_TYPE_NO_MORE {
		return h.bodyReadCloser.Close() // close due to no more data.
	}
	if _, err := h.bodyReadCloser.Write(data.Data); err != nil {
		return err
	}
	return nil
}

func (h *HttpProxyEst) Close(tell bool) error {
	return h.bodyReadCloser.Close() // close from client
}

func (h *HttpProxyEst) establish(s *Server, id string, addr string, header []byte) error {
	if header == nil {
		s.tellClosed(id)
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		_ = s.WsClient.WriteProxyMessage(ctx, id, pb.PROXY_DATA_TYPE_ESTABLISH_ERROR, nil)
		return errors.New("http header empty")
	}

	closed := make(chan bool)
	client := make(chan ClientData, 2) // for http at most 2 data buffers are needed(http body, TagNoMore tag).
	defer close(closed)
	defer close(client)

	s.addNewProxy(&ProxyServer{Id: id, ProxyIns: h})
	defer s.RemoveProxy(id)
	defer func() {
		if !h.bodyReadCloser.isClosed() { // if it is not closed by client.
			s.tellClosed(id) // todo
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := s.WsClient.WriteProxyMessage(ctx, id, pb.PROXY_DATA_TYPE_ESTABLISH_OK, nil); err != nil {
		return err
	}

	// get http request by header bytes.
	bufferHeader := bufio.NewReader(bytes.NewBuffer(header))
	req, err := http.ReadRequest(bufferHeader)
	if err != nil {
		return err
	}
	req.Body = h.bodyReadCloser

	// read request and copy response back
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return fmt.Errorf("transport error: %w", err)
	}
	defer resp.Body.Close()

	writer := NewWebSocketWriter(s.WsClient, id, context.Background())
	var headerBuffer bytes.Buffer
	util.HttpRespHeader(&headerBuffer, resp)
	writer.Write(headerBuffer.Bytes())
	if _, err := io.Copy(writer, resp.Body); err != nil {
		return fmt.Errorf("http body copy error: %w", err)
	}
	return nil
}

// tell the client the connection has been closed
func (s *Server) tellClosed(id string) error {
	// send finish flag to client
	base := &pb.Base{
		Type:    pb.MessageType_PROXY_CLOSE,
		Session: id,
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
