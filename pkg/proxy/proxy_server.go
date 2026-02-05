package proxy

import (
	"context"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"gaiasec-nodeagent/pkg/pb"
	"time"
)

type ProxyServer struct {
	Id      string // id of proxy connection
	done    chan ChanDone
	tcpConn net.Conn
}

func (ps *ProxyServer) onData(data ClientData) error {
	if _, err := ps.tcpConn.Write(data.Data); err != nil {
		ps.done <- ChanDone{true, err}
	}
	return nil
}

func (ps *ProxyServer) Close(tell bool) error {
	ps.done <- ChanDone{tell, ConnCloseByClient}
	return nil // todo error
}

// data: data send in establish step (can be nil).
func (ps *ProxyServer) establish(s *Server, addr string, source string, destination string) error {
	conn, err := net.DialTimeout("tcp", addr, time.Second*8) // todo config timeout
	if err != nil {
		return err
	}
	ps.tcpConn = conn
	defer conn.Close()

	ps.done = make(chan ChanDone, 2)
	//defer close(done)

	// todo check exists
	s.addNewProxyServer(ps)
	defer s.removeProxyServer(ps.Id)

	bytes := []byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	m := &pb.ProxyData{
		ProxyDataType: pb.PROXY_DATA_TYPE_DATA,
		Data:          bytes,
	}

	err = s.WsClient.SendMessage(m, pb.MessageType_PROXY_DATA_TO_CLIENT, source, destination, ps.Id)
	if err != nil {
		return err
	}

	go func() {
		writer := NewWSWriter(s.WsClient, ps.Id, context.Background(), source, destination, pb.MessageType_PROXY_DATA_TO_CLIENT)
		if _, err := io.Copy(writer, conn); err != nil {
			log.Debug("copy error,", err)
			ps.done <- ChanDone{true, err}
		}
		ps.done <- ChanDone{true, nil}
	}()

	d := <-ps.done
	// s.removeProxyServer(proxy.Id)
	// tellClosed is called outside this func.
	return d.err
}
