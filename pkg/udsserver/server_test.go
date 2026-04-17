package udsserver

import (
	"encoding/binary"
	"io"
	"net"
	"sync"
	"testing"

	"gaiasec-nodeagent/pkg/pb"
	"google.golang.org/protobuf/proto"
)

func TestServerHandleMessageRoutesToRegisteredAgent(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	server := &Server{
		Agent2SocketMap: make(map[string]*Client),
	}
	client := &Client{
		conn: &serverConn,
	}
	server.registerAgent("agent-1", client)

	message := &pb.Base{
		Type:        pb.MessageType_EXECUTE_COMMAND_REQUEST,
		Source:      "src",
		Destination: "agent-1",
		Session:     "session-1",
		Data:        []byte("payload"),
	}

	var (
		got []byte
		err error
		wg  sync.WaitGroup
	)
	wg.Add(1)
	go func() {
		defer wg.Done()

		lengthBuf := make([]byte, 4)
		if _, err = io.ReadFull(clientConn, lengthBuf); err != nil {
			return
		}
		payloadLen := binary.BigEndian.Uint32(lengthBuf)
		got = make([]byte, payloadLen)
		_, err = io.ReadFull(clientConn, got)
	}()

	server.HandleMessage(message)
	wg.Wait()

	if err != nil {
		t.Fatalf("read forwarded message: %v", err)
	}

	decoded := &pb.Base{}
	if err := proto.Unmarshal(got, decoded); err != nil {
		t.Fatalf("unmarshal forwarded message: %v", err)
	}
	if decoded.GetDestination() != "agent-1" {
		t.Fatalf("unexpected destination: %s", decoded.GetDestination())
	}
	if string(decoded.GetData()) != "payload" {
		t.Fatalf("unexpected payload: %q", string(decoded.GetData()))
	}
}

func TestServerAgentMapConcurrentAccess(t *testing.T) {
	server := &Server{
		Agent2SocketMap: make(map[string]*Client),
	}

	const goroutines = 16
	const iterations = 500

	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			client := &Client{}
			for i := 0; i < iterations; i++ {
				server.registerAgent(id, client)
				if got, ok := server.getAgentClient(id); ok && got != client {
					t.Errorf("unexpected client for %s", id)
				}
				_ = server.snapshotAgents()
				server.unregisterAgent(id)
			}
		}("agent-" + string(rune('a'+g)))
	}
	wg.Wait()
}
