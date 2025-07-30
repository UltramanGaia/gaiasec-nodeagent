package proxy

import (
	"context"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"sothoth-nodeagent/pkg/pb"
	"time"
)

type ProxyServer struct {
	Id      string // id of proxy connection
	done    chan ChanDone
	tcpConn net.Conn
}

func (e *ProxyServer) onData(data ClientData) error {
	if _, err := e.tcpConn.Write(data.Data); err != nil {
		e.done <- ChanDone{true, err}
	}
	return nil
}

func (e *ProxyServer) Close(tell bool) error {
	e.done <- ChanDone{tell, ConnCloseByClient}
	return nil // todo error
}

// data: data send in establish step (can be nil).
func (e *ProxyServer) establish(s *Server, addr string, source string, destination string) error {
	conn, err := net.DialTimeout("tcp", addr, time.Second*8) // todo config timeout
	if err != nil {
		return err
	}
	e.tcpConn = conn
	defer conn.Close()

	e.done = make(chan ChanDone, 2)
	//defer close(done)

	// todo check exists
	s.addNewProxyServer(e)
	defer s.removeProxyServer(e.Id)

	bytes := []byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	m := &pb.ProxyData{
		ProxyDataType: pb.PROXY_DATA_TYPE_DATA,
		Data:          bytes,
	}

	err = s.WsClient.SendMessage(m, pb.MessageType_PROXY_DATA_TO_CLIENT, source, destination, e.Id)
	if err != nil {
		return err
	}

	go func() {
		writer := NewWebSocketWriter(s.WsClient, e.Id, context.Background(), source, destination)
		if _, err := io.Copy(writer, conn); err != nil {
			log.Debug("copy error,", err)
			e.done <- ChanDone{true, err}
		}
		e.done <- ChanDone{true, nil}
	}()

	d := <-e.done
	// s.removeProxyServer(proxy.Id)
	// tellClosed is called outside this func.
	return d.err
}
