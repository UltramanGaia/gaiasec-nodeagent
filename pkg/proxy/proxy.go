package proxy

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"nhooyr.io/websocket/wspb"
	"sothoth-nodeagent/pkg/pb"
	"strconv"
	"time"
)

// proxy client handle one connection, send data to proxy server vai websocket.
type ProxyClient struct {
	Id          string
	Source      string
	Destination string
	onData      func(string, ServerData) // data from server todo data with  type
	onClosed    func(string, bool)       // close connection, param bool: do tellClose if true
	onError     func(string, error)      // if there are error messages
}

type ServerData struct {
	Tag  pb.PROXY_DATA_TYPE
	Data []byte
}

// tell wssocks proxy server to establish a proxy connection by sending server
// proxy address, type, initial data.
func (p *ProxyClient) Establish(client *Client, addr string) error {
	m := &pb.ProxyEstablishMessage{
		Addr:     addr,
		WithData: false,
	}
	data, err := proto.Marshal(m)
	if err != nil {
		return err
	}

	base := &pb.Base{
		Type:        pb.MessageType_PROXY_ESTABLISH,
		Source:      p.Source,
		Destination: p.Destination,
		Session:     p.Id,
		Data:        data,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	err = wspb.Write(ctx, client.server.WsClient.Conn, base)
	if err != nil {
		return err
	}

	return nil
}

type Socks5Client struct {
}

func (client *Socks5Client) Trigger(data []byte) bool {
	return len(data) >= 2 && data[0] == 0x05
}

// parsing socks5 header, and return address and parsing error
func (client *Socks5Client) ParseHeader(conn net.Conn, header []byte) (string, string, error) {
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
