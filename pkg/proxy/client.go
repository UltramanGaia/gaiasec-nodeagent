package proxy

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"nhooyr.io/websocket/wspb"
	"sothoth-nodeagent/pkg/config"
	"sothoth-nodeagent/pkg/pb"
	"sothoth-nodeagent/pkg/util"
	"strconv"
	"sync"
	"time"
)

type Client struct {
	conn   *net.TCPConn
	server *Server

	proxyMu   sync.RWMutex // mutex to operate proxies map.
	writeLock sync.RWMutex
}

func NewClient(conn *net.TCPConn, s *Server) (*Client, error) {
	return &Client{
		conn:   conn,
		server: s,
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

func parseHeader(conn net.Conn) (string, string, error) {
	// response to socks5 client
	// see rfc 1982 for more details (https://tools.ietf.org/html/rfc1928)
	n, err := conn.Write([]byte{0x05, 0x02}) // version and USERNAME/PASSWORD
	if err != nil {
		return "", "", err
	}

	// step 1
	var buffer [1024]byte
	n, err = conn.Read(buffer[:])
	if err != nil {
		return "", "", err
	}
	//method := buffer[0]
	userLen := int(buffer[1])
	username := string(buffer[2 : userLen+2])
	passLen := int(buffer[userLen+2])
	password := string(buffer[userLen+3 : userLen+3+passLen])
	log.Infof("username: %s, password: %s", username, password)

	// see rfc 1982 for more details (https://tools.ietf.org/html/rfc1928)
	n, err = conn.Write([]byte{0x05, 0x00}) // version and no authentication required
	if err != nil {
		return "", "", err
	}

	// step2: process client Requests and does Reply
	/**
	  +----+-----+-------+------+----------+----------+
	  |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
	  +----+-----+-------+------+----------+----------+
	  | 1  |  1  | X'00' |  1   | Variable |    2     |
	  +----+-----+-------+------+----------+----------+
	*/

	n, err = conn.Read(buffer[:])
	if err != nil {
		return "", "", err
	}
	if n < 6 {
		return "", "", errors.New("not a socks protocol")
	}

	var host string
	switch buffer[3] {
	case 0x01:
		// ipv4 address
		ipv4 := make([]byte, 4)
		if _, err := io.ReadAtLeast(bytes.NewReader(buffer[4:]), ipv4, len(ipv4)); err != nil {
			return "", "", err
		}
		host = net.IP(ipv4).String()
	case 0x04:
		// ipv6
		ipv6 := make([]byte, 16)
		if _, err := io.ReadAtLeast(bytes.NewReader(buffer[4:]), ipv6, len(ipv6)); err != nil {
			return "", "", err
		}
		host = net.IP(ipv6).String()
	case 0x03:
		// domain
		addrLen := int(buffer[4])
		domain := make([]byte, addrLen)
		if _, err := io.ReadAtLeast(bytes.NewReader(buffer[5:]), domain, addrLen); err != nil {
			return "", "", err
		}
		host = string(domain)
	}

	port := make([]byte, 2)
	err = binary.Read(bytes.NewReader(buffer[n-2:n]), binary.BigEndian, &port)
	if err != nil {
		return "", "", err
	}

	return username, net.JoinHostPort(host, strconv.Itoa((int(port[0])<<8)|int(port[1]))), nil
}

// parse target address and proxy type, and response to socks5/https client
func (client *Client) Reply(conn net.Conn) (string, string, error) {
	var buffer [1024]byte
	_, err := conn.Read(buffer[:])
	if err != nil {
		return "", "", err
	}

	// set address and type
	username, proxyAddr, err := parseHeader(conn)
	if err != nil {
		return "", "", err
	}

	return username, proxyAddr, nil
}

// create a new proxy with unique id
func (client *Client) NewProxy(username string, onData func(string, ServerData),
	onClosed func(string, bool), onError func(string, error)) *ProxyClient {
	id := util.GenerateID()
	cfg := config.GetInstance()
	proxyInstance := ProxyClient{Id: id, Source: cfg.NodeID, Destination: username, onData: onData, onClosed: onClosed, onError: onError}

	client.proxyMu.Lock()
	defer client.proxyMu.Unlock()

	client.server.proxies[id] = &proxyInstance
	return &proxyInstance
}

func (client *Client) transData(conn *net.TCPConn, username string, addr string) error {
	type Done struct {
		tell bool
		err  error
	}
	done := make(chan Done, 2)

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
	if proxyInstance, ok := client.server.proxies[id]; ok {
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
}

// remove current proxy by id
func (client *Client) RemoveProxy(id string) {
	client.proxyMu.Lock()
	defer client.proxyMu.Unlock()
	if _, ok := client.server.proxies[id]; ok {
		delete(client.server.proxies, id)
	}
}

type ServerData struct {
	Tag  pb.PROXY_DATA_TYPE
	Data []byte
}
