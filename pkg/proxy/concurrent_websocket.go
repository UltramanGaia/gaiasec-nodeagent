package proxy

import (
	"context"
	"nhooyr.io/websocket"
	"sothoth-nodeagent/pkg/pb"
	"sothoth-nodeagent/pkg/wsclient"
	"sync"
)

type ConcurrentWebSocketInterface interface {
	WSClose() error
	WriteWSJSON(data interface{}) error
}

// add lock to websocket connection to make sure only one goroutine can write this websocket.
type ConcurrentWebSocket struct {
	WsConn *websocket.Conn
}

// close websocket connection
func (wsc *ConcurrentWebSocket) WSClose() error {
	return wsc.WsConn.Close(websocket.StatusNormalClosure, "")
}

type webSocketWriter struct {
	Client      *wsclient.Client
	Id          string // connection id.
	Source      string
	Destination string
	Ctx         context.Context
	Type        int // type of trans data.
	Mu          *sync.Mutex
}

func NewWebSocketWriter(client *wsclient.Client, id string, ctx context.Context, source string, destination string) *webSocketWriter {
	return &webSocketWriter{Client: client, Id: id, Ctx: ctx, Source: source, Destination: destination}
}

func NewWebSocketWriterWithMutex(client *wsclient.Client, id string, ctx context.Context, source string, destination string) *webSocketWriter {
	return &webSocketWriter{Client: client, Id: id, Ctx: ctx, Mu: &sync.Mutex{}, Source: source, Destination: destination}
}

func (writer *webSocketWriter) CloseWsWriter(cancel context.CancelFunc) {
	if writer.Mu != nil {
		writer.Mu.Lock()
		defer writer.Mu.Unlock()
	}
	cancel()
}

func (writer *webSocketWriter) Write(buffer []byte) (n int, err error) {
	if writer.Mu != nil {
		writer.Mu.Lock()
		defer writer.Mu.Unlock()
	}
	// make sure context is not Canceled/DeadlineExceeded before Write.
	if writer.Ctx.Err() != nil {
		return 0, writer.Ctx.Err()
	}
	if err := writer.Client.WriteProxyMessage(writer.Ctx, writer.Id, pb.PROXY_DATA_TYPE_DATA, buffer, writer.Source, writer.Destination); err != nil {
		return 0, err
	} else {
		return len(buffer), nil
	}
}
