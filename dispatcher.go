package main

import (
	"context"

	pb "github.com/fieldkit/app-protocol"
)

type apiHandler func(ctx context.Context, wireQuery *pb.WireMessageQuery) (reply *pb.WireMessageReply, err error)

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
