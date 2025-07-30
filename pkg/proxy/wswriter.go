package proxy

import (
	"context"
	"sothoth-nodeagent/pkg/pb"
	"sothoth-nodeagent/pkg/wsclient"
)

type WSWriter struct {
	Client      *wsclient.Client
	Id          string // connection id.
	Source      string
	Destination string
	Ctx         context.Context
	Type        int // type of trans data.
	MsgType     pb.MessageType
}

func NewWSWriter(client *wsclient.Client, id string, ctx context.Context, source string, destination string, msgType pb.MessageType) *WSWriter {
	return &WSWriter{Client: client, Id: id, Ctx: ctx, Source: source, Destination: destination, MsgType: msgType}
}

func (writer *WSWriter) CloseWsWriter(cancel context.CancelFunc) {
	cancel()
}

func (writer *WSWriter) Write(buffer []byte) (n int, err error) {
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
