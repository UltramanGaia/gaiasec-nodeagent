package udsserver

import (
	"encoding/binary"
	"fmt"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
	"gaiasec-nodeagent/pkg/constant"
	"gaiasec-nodeagent/pkg/pb"
	"gaiasec-nodeagent/pkg/util"
	"sync"
)

type Client struct {
	conn      *net.Conn
	server    *Server
	writeLock sync.RWMutex
	agentId   string
	buffer    []byte // 缓存不完整的数据包
}

func NewClient(conn *net.Conn, s *Server) (*Client, error) {
	return &Client{
		conn:    conn,
		server:  s,
		agentId: "",
		buffer:  make([]byte, 0),
	}, nil
}

// nodeagent下面的agent发送消息给其他agent，上报Server即可
func (c *Client) HandleAgentMessage() {
	defer func() {
		c.unregister(c.agentId)
	}()

	for {
		data, err := c.ReadMessage()
		if err != nil {
			log.Error("Read message error: ", err)
			return
		}

		if c.agentId == "" {
			// 解析Protobuf消息
			msg := &pb.Base{}
			if err := proto.Unmarshal(data, msg); err != nil {
				log.Error("Unmarshal message error: ", err)
			}
			if msg.GetType() != pb.MessageType_REGISTER { // 第一个消息必须是登录消息
				log.Error("first message must be register message")
			}
			registerMsg := &pb.Register{}
			if err := proto.Unmarshal(msg.GetData(), registerMsg); err != nil {
				log.Error("first message parse error: ", err)
				return
			}
			c.agentId = registerMsg.Id
			c.server.Agent2SocketMap[c.agentId] = c
			log.Infof("Agent %s register", registerMsg.Id)
		}

		log.Debug("Received message from agent, send to server")
		// 直接将Agent侧收到的消息转发给Server即可
		_ = c.server.WsClient.SendMessageBytes(data)
	}
}

func (c *Client) ReadMessage() ([]byte, error) {
	for {
		// 读取数据到临时缓冲区
		temp := make([]byte, 1024)
		n, err := (*c.conn).Read(temp)
		if err != nil {
			if err != io.EOF {
				log.Errorf("read data error: %v\n", err)
			} else {
				log.Errorf("client %s close\n", (*c.conn).RemoteAddr())
			}
			return nil, err
		}

		// 将新读取的数据追加到缓冲区
		c.buffer = append(c.buffer, temp[:n]...)

		// 处理缓冲区中的完整消息
		for {
			// 检查是否有足够的数据解析长度前缀
			if len(c.buffer) < 4 {
				break
			}

			// 解析长度前缀(大端字节序)
			messageLength := binary.BigEndian.Uint32(c.buffer[:4])

			// 验证消息长度是否合法，最大长度设置为100MB，避免恶意攻击
			if messageLength <= 0 || messageLength > 100*1024*1024 {
				return nil, fmt.Errorf("invalid message length: %d", messageLength)
			}

			// 检查是否有完整的消息数据
			totalLength := 4 + int(messageLength)
			if len(c.buffer) < totalLength {
				break // 数据不完整，等待更多数据
			}

			// 提取消息体
			messageData := c.buffer[4:totalLength]
			// 保留缓冲区中剩余的数据
			c.buffer = c.buffer[totalLength:]

			return messageData, nil
		}
	}
}

func (c *Client) unregister(agentId string) {
	if agentId != "" {
		log.Infof("Agent %s logout", agentId)
		agentLogout := &pb.Unregister{
			Id: agentId,
		}

		err := c.server.WsClient.SendMessage(agentLogout, pb.MessageType_UNREGISTER, agentId, constant.SERVER_ID, util.GenerateID())
		if err != nil {
			log.Error("Emit logout error: ", err)
		}
		delete(c.server.Agent2SocketMap, agentId)
	}
	_ = (*c.conn).Close()
}

func (c *Client) sendMessage(data []byte) error {
	c.writeLock.Lock()
	defer c.writeLock.Unlock()

	// 创建一个4字节的缓冲区存储长度
	lengthBuf := make([]byte, 4)
	// 将数据长度转换为大端序的4字节
	binary.BigEndian.PutUint32(lengthBuf, uint32(len(data)))

	_, err := (*c.conn).Write(lengthBuf)
	if err != nil {
		return err
	}
	_, err = (*c.conn).Write(data)
	return err
}

func (c *Client) SendMessage(message *pb.Base) error {
	data, err := proto.Marshal(message)
	if err != nil {
		return err
	}
	return c.sendMessage(data)
}
