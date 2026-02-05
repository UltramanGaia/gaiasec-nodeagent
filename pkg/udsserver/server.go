package udsserver

import (
	log "github.com/sirupsen/logrus"
	"net"
	"os"
	"os/signal"
	"gaiasec-nodeagent/pkg/config"
	"gaiasec-nodeagent/pkg/pb"
	"gaiasec-nodeagent/pkg/wsclient"
	"syscall"
)

type Server struct {
	socketPath      string
	WsClient        *wsclient.Client
	Agent2SocketMap map[string]*Client
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
	var socketPath string
	socketPath = "/gaiasec/nodeagent.sock"
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
		if client, ok := s.Agent2SocketMap[destination]; ok {
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
