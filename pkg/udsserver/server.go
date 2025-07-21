package udsserver

import (
	log "github.com/sirupsen/logrus"
	"net"
	"os"
	"os/signal"
	"sothoth-nodeagent/pkg/config"
	"sothoth-nodeagent/pkg/wsclient"
	"syscall"
)

type Server struct {
	socketPath      string
	wsClient        *wsclient.Client
	Agent2SocketMap map[string]net.Conn
	listener        net.Listener
	running         bool
	cfg             *config.Config
}

func NewServer(client *wsclient.Client) (*Server, error) {
	cfg := config.GetInstance()

	socketPath := cfg.SothothDir + "/nodeagent.sock"
	return &Server{
		socketPath:      socketPath,
		wsClient:        client,
		Agent2SocketMap: make(map[string]net.Conn),
	}, nil
}

func (s Server) Start() {

}

func (s Server) Stop() {
	s.running = false
	if s.listener != nil {
		s.listener.Close()
		s.listener = nil
	}
}

func (s Server) handleAgentMessage() {
	server := startUnixSocketServer()
	for {
		conn := accept(server)
		if conn == nil {
			continue
		}

		//go processAgentMessage(conn, s.wsClient)

	}
}

var agent2SocketMap map[string]net.Conn

func init() {
	agent2SocketMap = make(map[string]net.Conn)
}

func startUnixSocketServer() net.Listener {
	var socketPath string
	socketPath = "/sothoth/nodeagent.sock"
	log.Infof("Starting unix socket server at %s", socketPath)
	_ = os.Remove(socketPath)
	server, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Errorf("Error starting unix socket server: %s", err)
		os.Exit(1)
	}
	err = os.Chmod(socketPath, 0666)
	if err != nil {
		log.Errorf("Error changing socket permissions: %s", err)
		os.Exit(1)
	}
	log.Infof("Started unix socket server at %s", socketPath)

	// handle common process-killing signals
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, os.Kill, syscall.SIGTERM)
	go func(c chan os.Signal) {
		sig := <-c
		log.Infof("Received signal: %s", sig)
		_ = server.Close()
		os.Exit(0)
	}(signals)
	return server

}

func accept(listener net.Listener) net.Conn {
	conn, err := listener.Accept()
	if err != nil {
		return nil
	}
	log.Infof("Accepted connection from %s", conn.RemoteAddr())
	return conn
}

//func processAgentMessage(conn net.Conn, client *wsclient.Client) {
//	defer logout(conn, client)
//	reader := bufio.NewReader(conn)
//	agentId, err := login(reader, client)
//	if err != nil {
//		log.Errorf("Auth error: %s", err)
//		return
//	}
//	log.Infof("Agent %s login", agentId)
//	agent2SocketMap[agentId] = conn
//	for {
//		message, err := reader.ReadString('\n')
//		if err != nil {
//			if err.Error() == "EOF" {
//				log.Errorf("Agent %s lost connection", agentId)
//			}
//			break
//		}
//		data := message[:len(message)-1]
//		log.Debug("push: ", data)
//		err = client.emit(model.PushEvent, data)
//		if err != nil {
//			log.Error("Emit push error: ", err)
//		}
//	}
//
//}

//func login(reader *bufio.Reader, na *NodeAgent) (string, error) {
//	msg, err := reader.ReadString('\n')
//	if err != nil {
//		return "", err
//	}
//	msg = msg[:len(msg)-1]
//	var info model.LoginResp
//	err = json.Unmarshal([]byte(msg), &info)
//	if err != nil {
//		return "", err
//	}
//	return info.AgentId, nil
//}
//
//func logout(conn net.Conn, na *NodeAgent) {
//	agentId := ""
//	for k, v := range agent2SocketMap {
//		if v == conn {
//			agentId = k
//			break
//		}
//	}
//	if agentId == "" {
//		log.Infof("Agent %s logout", agentId)
//		msg := &model.LogoutResp{
//			AgentId: agentId,
//		}
//		err := na.emit(model.LogoutEvent, msg)
//		if err != nil {
//			log.Error("Emit logout error: ", err)
//		}
//		delete(agent2SocketMap, agentId)
//	}
//	_ = conn.Close()
//}
