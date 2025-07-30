package proxy

import (
	"context"
	"sothoth-nodeagent/pkg/pb"
	"sothoth-nodeagent/pkg/wsclient"
	"sync"
)

type WebSocketWriter struct {
	Client      *wsclient.Client
	Id          string // connection id.
	Source      string
	Destination string
	Ctx         context.Context
	Type        int // type of trans data.
	Mu          *sync.Mutex
	MsgType     pb.MessageType
}

func NewWebSocketWriter(client *wsclient.Client, id string, ctx context.Context, source string, destination string, msgType pb.MessageType) *WebSocketWriter {
	return &WebSocketWriter{Client: client, Id: id, Ctx: ctx, Source: source, Destination: destination, MsgType: msgType}
}

func NewWebSocketWriterWithMutex(client *wsclient.Client, id string, ctx context.Context, source string, destination string, msgType pb.MessageType) *WebSocketWriter {
	return &WebSocketWriter{Client: client, Id: id, Ctx: ctx, Mu: &sync.Mutex{}, Source: source, Destination: destination, MsgType: msgType}
}

func (writer *WebSocketWriter) CloseWsWriter(cancel context.CancelFunc) {
	//if writer.Mu != nil {
	//	writer.Mu.Lock()
	//	defer writer.Mu.Unlock()
	//}
	cancel()
}

func (writer *WebSocketWriter) Write(buffer []byte) (n int, err error) {
	//if writer.Mu != nil {
	//	writer.Mu.Lock()
	//	defer writer.Mu.Unlock()
	//}
	// make sure context is not Canceled/DeadlineExceeded before Write.
	if writer.Ctx.Err() != nil {
		return 0, writer.Ctx.Err()
	}

	m := &pb.ProxyData{
		ProxyDataType: pb.PROXY_DATA_TYPE_DATA,
		Data:          buffer,
	}

	err = writer.Client.SendMessage(m, writer.MsgType, writer.Source, writer.Destination, writer.Id)
	if err != nil {
		return 0, err
	} else {
		return len(buffer), nil
	}
}
