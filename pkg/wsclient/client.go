package wsclient

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wspb"
	"os"
	"sothoth-nodeagent/pkg/config"
	"sothoth-nodeagent/pkg/pb"
	"sync"
	"time"
)

type Client struct {
	ctx              context.Context
	cancel           context.CancelFunc
	uri              string
	Conn             *websocket.Conn
	writeLock        sync.RWMutex
	id               int
	namespace        string
	retryNum         int
	pauseBeforeRetry int
	Running          bool
	cfg              *config.Config
}

func NewClient(uri string, retryNum int, pauseBeforeRetry int) (client *Client, err error) {
	log.Info("try to connect server: [" + uri + "]")
	ctx, cancel := context.WithCancel(context.Background())

	client = &Client{
		ctx:              ctx,
		cancel:           cancel,
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
	log.Errorf("trying to reconnect client: [" + c.uri + "]")
	for i := 0; c.retryNum == 0 || i <= c.retryNum; i++ {
		time.Sleep(time.Duration(c.pauseBeforeRetry) * time.Second)
		client, _, err := websocket.Dial(c.ctx, c.uri, nil)
		if err == nil {
			log.Info("Reconnect success")
			c.Conn = client
			return nil
		}
	}
	return fmt.Errorf("max retry exceeded")
}

func (c *Client) Start() {
	c.Running = true
	conn, _, err := websocket.Dial(c.ctx, c.uri, nil)
	if err != nil {
		log.Error("dial:", err)
		os.Exit(-1)
	}
	// read messages from webSocket
	conn.SetReadLimit(1 << 25)
	c.Conn = conn
}

func (c *Client) Stop() {
	c.Running = false
	c.cancel()
	if c.Conn != nil {
		_ = c.Conn.Close(websocket.StatusNormalClosure, "")
		c.Conn = nil
	}
}

func (c *Client) SendMessageBytes(data []byte) error {
	//c.writeLock.Lock()
	//defer c.writeLock.Unlock()
	return c.Conn.Write(c.ctx, websocket.MessageBinary, data)
}

func (c *Client) SendMessage(m proto.Message, msgType pb.MessageType, source string, destination string, sessionId string) error {
	data, err := proto.Marshal(m)
	if err != nil {
		return err
	}

	msg := pb.Base{
		Type:        msgType,
		Source:      source,
		Destination: destination,
		Session:     sessionId,
		Data:        data,
	}

	err = wspb.Write(c.ctx, c.Conn, &msg)
	return err
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

	err = wspb.Write(c.ctx, c.Conn, &msg)
	return err
}

func (c *Client) ReadMessage(v *pb.Base) (err error) {
	//msg := pb.Base{}
	return wspb.Read(c.ctx, c.Conn, v)
}
