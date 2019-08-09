package main

import (
	"context"

	pb "github.com/fieldkit/app-protocol"
)

type ReplyWriter interface {
	Prepare(size int) error
	WriteReply(reply *pb.HttpReply) (int, error)
	WriteBytes(bytes []byte) (int, error)
}

type ApiHandler func(ctx context.Context, device *FakeDevice, query *pb.HttpQuery, reply ReplyWriter) (err error)

type Dispatcher struct {
	handlers map[pb.QueryType]ApiHandler
}

func NewDispatcher() *Dispatcher {
	handlers := make(map[pb.QueryType]ApiHandler)
	return &Dispatcher{
		handlers: handlers,
	}
}

func (rd *Dispatcher) AddHandler(qt pb.QueryType, handler ApiHandler) {
	rd.handlers[qt] = handler
}
