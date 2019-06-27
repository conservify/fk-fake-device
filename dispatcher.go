package main

import (
	"context"

	pb "github.com/fieldkit/app-protocol"
)

type replyWriter interface {
	Prepare(size int) error
	WriteReply(reply *pb.WireMessageReply) (int, error)
	WriteBytes(bytes []byte) (int, error)
}

type apiHandler func(ctx context.Context, device *FakeDevice, query *pb.WireMessageQuery, reply replyWriter) (err error)

type dispatcher struct {
	handlers map[pb.QueryType]apiHandler
}

func newDispatcher() *dispatcher {
	handlers := make(map[pb.QueryType]apiHandler)
	return &dispatcher{
		handlers: handlers,
	}
}

func (rd *dispatcher) AddHandler(qt pb.QueryType, handler apiHandler) {
	rd.handlers[qt] = handler
}
