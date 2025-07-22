package udsserver

import (
	"encoding/binary"
	"fmt"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
	"os"
	"os/signal"
	"sothoth-nodeagent/pkg/config"
	"sothoth-nodeagent/pkg/pb"
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

func (s *Server) Start() {
	s.running = true
	var socketPath string
	socketPath = "/sothoth/nodeagent.sock"
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
		go processAgentMessage(conn, s.wsClient)
	}
}

var agent2SocketMap map[string]net.Conn

func init() {
	agent2SocketMap = make(map[string]net.Conn)
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

func processAgentMessage(conn net.Conn, client *wsclient.Client) {
	var agentId string
	defer func() {
		if agentId != "" {
			log.Infof("Agent %s logout", agentId)
			agentLogout := pb.AgentLogout{
				AgentId: agentId,
			}

			err := client.Send(pb.MessageType_AGENT_LOGOUT, &agentLogout)
			if err != nil {
				log.Error("Emit logout error: ", err)
			}
			delete(agent2SocketMap, agentId)
		}
		_ = conn.Close()
	}()
	// 用于缓存不完整的数据包
	buffer := make([]byte, 0)

	for {
		// 读取数据到临时缓冲区
		temp := make([]byte, 1024)
		n, err := conn.Read(temp)
		if err != nil {
			if err != io.EOF {
				fmt.Printf("读取数据错误: %v\n", err)
			} else {
				fmt.Printf("客户端 %s 断开连接\n", conn.RemoteAddr())
			}
			return
		}

		// 将新读取的数据追加到缓冲区
		buffer = append(buffer, temp[:n]...)

		// 处理缓冲区中的完整消息
		for {
			// 检查是否有足够的数据解析长度前缀
			if len(buffer) < 4 {
				break
			}

			// 解析长度前缀(大端字节序)
			messageLength := binary.BigEndian.Uint32(buffer[:4])

			// 验证消息长度是否合法，最大长度设置为100MB，避免恶意攻击
			if messageLength <= 0 || messageLength > 100*1024*1024 {
				fmt.Printf("无效的消息长度: %d，关闭连接\n", messageLength)
				return
			}

			// 检查是否有完整的消息数据
			totalLength := 4 + int(messageLength)
			if len(buffer) < totalLength {
				break // 数据不完整，等待更多数据
			}

			// 提取消息体
			messageData := buffer[4:totalLength]
			// 保留缓冲区中剩余的数据
			buffer = buffer[totalLength:]

			if agentId == "" {
				// 解析Protobuf消息
				msg := &pb.BaseMessage{}
				if err := proto.Unmarshal(messageData, msg); err != nil {
					return
				}
				if msg.GetType() != pb.MessageType_AGENT_LOGIN { // 第一个消息必须是登录消息
					return
				}
				loginMsg := &pb.AgentLogin{}
				if err := proto.Unmarshal(msg.GetData(), loginMsg); err != nil {
					return
				}
				agentId = loginMsg.AgentId
				agent2SocketMap[agentId] = conn
			}

			// 直接将Agent侧收到的消息转发给Server即可
			_ = client.SendMessage(messageData)
		}
	}
}
