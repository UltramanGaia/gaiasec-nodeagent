package wsclient

import (
	"fmt"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
	"os"
	"sothoth-nodeagent/pkg/config"
	"sothoth-nodeagent/pkg/pb"
	"sync"
	"time"
)

type Client struct {
	uri              string
	conn             *websocket.Conn
	writeLock        sync.RWMutex
	id               int
	namespace        string
	retryNum         int
	pauseBeforeRetry int
	running          bool
	cfg              *config.Config
}

func NewClient(uri string, retryNum int, pauseBeforeRetry int) (client *Client, err error) {
	log.Info("Try to connect server: [" + uri + "]")
	c, _, err := websocket.DefaultDialer.Dial(uri, nil)
	if err != nil {
		log.Error("dial:", err)
		os.Exit(-1)
	}
	c.SetReadLimit(0) // 禁用读超时

	client = &Client{
		uri:              uri,
		conn:             c,
		id:               0,
		retryNum:         retryNum,
		pauseBeforeRetry: pauseBeforeRetry,
		running:          false,
		cfg:              config.GetInstance(),
	}
	return client, nil
}

func (c *Client) Reconnect() error {
	if !c.running {
		return fmt.Errorf("it is not running")
	}
	log.Errorf("Trying to reconnect client: [" + c.uri + "]")
	for i := 0; c.retryNum == 0 || i <= c.retryNum; i++ {
		client, _, err := websocket.DefaultDialer.Dial(c.uri, nil)
		if err == nil {
			log.Info("Reconnect success")
			c.conn = client
			return nil
		}
		time.Sleep(time.Duration(c.pauseBeforeRetry) * time.Second)
	}
	return fmt.Errorf("max retry exceeded")
}

func (c *Client) Start() error {
	c.running = true

	// 连接建立成功，上报节点信息
	c.reportNodeLogin()

	for {
		messageType, message, err := c.conn.ReadMessage()
		if err != nil {
			if !c.running {
				return nil
			}
			log.Error("read:", err)
			err = c.Reconnect()
			if err != nil {
				log.Error("Reconnect failed")
				return err
			}
			continue
		}

		if messageType == websocket.BinaryMessage {
			// 解析基础消息
			baseMessage := &pb.BaseMessage{}
			if err := proto.Unmarshal(message, baseMessage); err != nil {
				log.Println("解析基础消息失败:", err)
				continue
			}
			// 根据消息类型处理
			switch baseMessage.Type {
			case pb.MessageType_HEARTBEAT:
				nodeHeartbeat := &pb.NodeHeartbeat{}
				if err := proto.Unmarshal(baseMessage.Data, nodeHeartbeat); err != nil {
					log.Println("解析登录请求失败:", err)
					continue
				}
			default:
				log.Println("未知消息类型")
			}
		}
	}
}

func (c *Client) Stop() {
	c.running = false
	c.conn.Close()
}

func (c *Client) sendMessage(data []byte) error {
	c.writeLock.Lock()
	defer c.writeLock.Unlock()
	w, err := c.conn.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	if err != nil {
		return err
	}
	return w.Close()
}

func (c *Client) Send(msgType pb.MessageType, m proto.Message) error {
	data, err := proto.Marshal(m)
	if err != nil {
		return err
	}

	msg := pb.BaseMessage{
		Type: msgType,
		Data: data,
	}

	bytes, err := proto.Marshal(&msg)
	if err != nil {
		return err
	}
	return c.sendMessage(bytes)
}
