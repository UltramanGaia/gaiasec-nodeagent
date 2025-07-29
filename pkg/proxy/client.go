package proxy

import (
	"context"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"nhooyr.io/websocket/wspb"
	"sothoth-nodeagent/pkg/config"
	"sothoth-nodeagent/pkg/pb"
	"sothoth-nodeagent/pkg/util"
	"sync"
	"time"
)

type Client struct {
	conn      *net.TCPConn
	server    *Server
	proxies   map[string]*ProxyClient // all proxies on this websocket.
	proxyMu   sync.RWMutex            // mutex to operate proxies map.
	writeLock sync.RWMutex
}

func NewClient(conn *net.TCPConn, s *Server) (*Client, error) {
	return &Client{
		conn:    conn,
		server:  s,
		proxies: make(map[string]*ProxyClient),
	}, nil
}

func (client *Client) HandleSocks5Conn() {
	// defer c.Close()
	defer client.conn.Close()
	// In reply, we can get proxy type, target address and first send data.
	username, addr, err := client.Reply(client.conn)
	if err != nil {
		log.Error("reply error: ", err)
	}

	// on connection established, copy data now.
	if err := client.transData(client.conn, username, addr); err != nil {
		log.Error("trans error: ", err)
	}
}

// parse target address and proxy type, and response to socks5/https client
func (client *Client) Reply(conn net.Conn) (string, string, error) {
	var buffer [1024]byte

	n, err := conn.Read(buffer[:])
	if err != nil {
		return "", "", err
	}

	proxyInstance := &Socks5Client{}
	// set address and type
	username, proxyAddr, err := proxyInstance.ParseHeader(conn, buffer[:n])
	if err != nil {
		return "", "", err
	}

	return username, proxyAddr, nil
}

// create a new proxy with unique id
func (client *Client) NewProxy(username string, onData func(string, ServerData),
	onClosed func(string, bool), onError func(string, error)) *ProxyClient {
	id := util.RenerateID()
	cfg := config.GetInstance()
	proxyInstance := ProxyClient{Id: id, Source: cfg.NodeID, Destination: username, onData: onData, onClosed: onClosed, onError: onError}

	client.proxyMu.Lock()
	defer client.proxyMu.Unlock()

	client.proxies[id] = &proxyInstance
	return &proxyInstance
}

func (client *Client) transData(conn *net.TCPConn, username string, addr string) error {
	type Done struct {
		tell bool
		err  error
	}
	done := make(chan Done, 2)
	// defer close(done)

	// create a with proxy with callback func
	proxyInstance := client.NewProxy(username, func(id string, data ServerData) {
		if _, err := conn.Write(data.Data); err != nil {
			done <- Done{true, err}
		}
	}, func(id string, tell bool) {
		done <- Done{tell, nil}
	}, func(id string, err error) {
		if err != nil {
			done <- Done{true, err}
		}
	})

	// tell server to establish connection
	if err := proxyInstance.Establish(client, addr); err != nil {
		client.RemoveProxy(proxyInstance.Id)
		err := client.TellClose(proxyInstance.Id, proxyInstance.Source, proxyInstance.Destination)
		if err != nil {
			log.Error("close error", err)
		}
		return err
	}

	// trans incoming data from proxy client application.
	ctx, cancel := context.WithCancel(context.Background())
	writer := NewWebSocketWriterWithMutex(client.server.WsClient, proxyInstance.Id, ctx, proxyInstance.Source, proxyInstance.Destination)
	go func() {
		_, err := io.Copy(writer, conn)
		if err != nil {
			log.Error("write error: ", err)
		}
		done <- Done{true, err}
	}()
	defer writer.CloseWsWriter(cancel) // cancel data writing

	d := <-done
	client.RemoveProxy(proxyInstance.Id)
	if d.tell {
		if err := client.TellClose(proxyInstance.Id, proxyInstance.Source, proxyInstance.Destination); err != nil {
			return err
		}
	}
	if d.err != nil {
		return d.err
	}
	return nil
}

func (client *Client) GetProxyById(id string) *ProxyClient {
	client.proxyMu.RLock()
	defer client.proxyMu.RUnlock()
	if proxyInstance, ok := client.proxies[id]; ok {
		return proxyInstance
	}
	return nil
}

// tell the remote proxy server to close this connection.
func (client *Client) TellClose(id string, source string, destination string) error {
	// send finish flag to client
	base := &pb.Base{
		Type:        pb.MessageType_PROXY_CLOSE,
		Source:      source,
		Destination: destination,
		Session:     id,
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	return wspb.Write(ctx, client.server.WsClient.Conn, base)

	return nil
}

// remove current proxy by id
func (client *Client) RemoveProxy(id string) {
	client.proxyMu.Lock()
	defer client.proxyMu.Unlock()
	if _, ok := client.proxies[id]; ok {
		delete(client.proxies, id)
	}
}
