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
	Running          bool
	cfg              *config.Config
}

func NewClient(uri string, retryNum int, pauseBeforeRetry int) (client *Client, err error) {
	log.Info("Try to connect server: [" + uri + "]")

	client = &Client{
		uri:              uri,
		id:               0,
		retryNum:         retryNum,
		pauseBeforeRetry: pauseBeforeRetry,
		Running:          false,
		cfg:              config.GetInstance(),
	}
	return client, nil
}

func (c *Client) Reconnect() error {
	if !c.Running {
		return fmt.Errorf("it is not Running")
	}
	log.Errorf("Trying to reconnect client: [" + c.uri + "]")
	for i := 0; c.retryNum == 0 || i <= c.retryNum; i++ {
		time.Sleep(time.Duration(c.pauseBeforeRetry) * time.Second)
		client, _, err := websocket.DefaultDialer.Dial(c.uri, nil)
		if err == nil {
			log.Info("Reconnect success")
			c.conn = client
			return nil
		}
	}
	return fmt.Errorf("max retry exceeded")
}

func (c *Client) Start() {
	c.Running = true
	conn, _, err := websocket.DefaultDialer.Dial(c.uri, nil)
	if err != nil {
		log.Error("dial:", err)
		os.Exit(-1)
	}
	conn.SetReadLimit(0) // 禁用读超时
	c.conn = conn
}

func (c *Client) Stop() {
	c.Running = false
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
}

func (c *Client) SendMessage(data []byte) error {
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

	msg := pb.Base{
		Type: msgType,
		Data: data,
	}

	bytes, err := proto.Marshal(&msg)
	if err != nil {
		return err
	}
	return c.SendMessage(bytes)
}

func (c *Client) ReadMessage() (messageType int, p []byte, err error) {
	return c.conn.ReadMessage()
}
