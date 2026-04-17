package udsserver

import (
	"gaiasec-nodeagent/pkg/config"
	"gaiasec-nodeagent/pkg/constant"
	"gaiasec-nodeagent/pkg/pb"
	"gaiasec-nodeagent/pkg/util"
	"gaiasec-nodeagent/pkg/wsclient"
	log "github.com/sirupsen/logrus"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type Server struct {
	socketPath      string
	WsClient        *wsclient.Client
	Agent2SocketMap map[string]*Client
	mu              sync.RWMutex
	listener        net.Listener
	running         bool
	cfg             *config.Config
}

func NewServer(client *wsclient.Client) (*Server, error) {
	cfg := config.GetInstance()

	socketPath := cfg.GaiaSecDir + "/nodeagent.sock"
	return &Server{
		socketPath:      socketPath,
		WsClient:        client,
		Agent2SocketMap: make(map[string]*Client),
	}, nil
}

func (s *Server) Start() {
	s.running = true
	cfg := config.GetInstance()

	var socketPath string
	socketPath = cfg.GaiaSecDir + "/nodeagent.sock"
	log.Infof("Starting unix socket listener at %s", socketPath)
	_ = os.Remove(socketPath)
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Errorf("Error starting unix socket listener: %s", err)
		os.Exit(1)
	}
	s.listener = listener
	err = os.Chmod(socketPath, 0666)
	if err != nil {
		log.Errorf("Error changing socket permissions: %s", err)
		os.Exit(1)
	}
	log.Infof("Started unix socket listener at %s", socketPath)

	// handle common process-killing signals
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, os.Kill, syscall.SIGTERM)
	go func(c chan os.Signal) {
		sig := <-c
		log.Infof("Received signal: %s", sig)
		_ = s.listener.Close()
		os.Exit(0)
	}(signals)
}

func (s *Server) Stop() {
	s.running = false
	if s.listener != nil {
		_ = s.listener.Close()
		s.listener = nil
	}
}

func (s *Server) HandleAgentMessage() {
	for {
		conn := s.accept()
		if conn == nil {
			continue
		}
		client, err := NewClient(&conn, s)
		if err != nil {
			log.Errorf("Error creating client: %s", err)
			continue
		}
		go client.HandleAgentMessage()
	}
}

func (s *Server) HandleMessage(message *pb.Base) {
	destination := message.Destination
	if destination != "" {
		if client, ok := s.getAgentClient(destination); ok {
			err := client.SendMessage(message)
			if err != nil {
				log.Errorf("Error writing to agent %s: %s", destination, err)
			}
		} else {
			log.Errorf("Agent %s not found", destination)
		}
	}
}

func (s *Server) accept() net.Conn {
	if s.listener == nil || !s.running {
		return nil
	}
	conn, err := s.listener.Accept()
	if err != nil {
		return nil
	}
	log.Infof("Accepted connection from %s", conn.RemoteAddr())
	return conn
}

func (s *Server) BroadcastAgentStatus() {
	for agentId, client := range s.snapshotAgents() {
		if client.registerMsg != nil {
			log.Infof("Broadcast agent status for %s", agentId)
			err := s.WsClient.SendMessage(client.registerMsg, pb.MessageType_REGISTER, agentId, constant.SERVER_ID, util.GenerateID())
			if err != nil {
				log.Errorf("Failed to broadcast agent status for %s: %v", agentId, err)
			}
		}
	}
}

func (s *Server) registerAgent(agentID string, client *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Agent2SocketMap[agentID] = client
}

func (s *Server) unregisterAgent(agentID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Agent2SocketMap, agentID)
}

func (s *Server) getAgentClient(agentID string) (*Client, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	client, ok := s.Agent2SocketMap[agentID]
	return client, ok
}

func (s *Server) snapshotAgents() map[string]*Client {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot := make(map[string]*Client, len(s.Agent2SocketMap))
	for agentID, client := range s.Agent2SocketMap {
		snapshot[agentID] = client
	}
	return snapshot
}
