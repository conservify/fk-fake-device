package main

import (
	pb "github.com/fieldkit/app-protocol"
	"github.com/golang/protobuf/proto"
	"log"
	"net"
)

type rpcContext struct {
	c net.Conn
}

type rpcHandler func(rc *rpcContext, wireQuery *pb.WireMessageQuery) error

func (rc *rpcContext) writeError(message string) error {
	wireReply := &pb.WireMessageReply{
		Type: pb.ReplyType_REPLY_ERROR,
		Errors: []*pb.Error{
			&pb.Error{
				Message: message,
			},
		},
	}
	rc.writeMessage(wireReply)

	return nil
}

func (rc *rpcContext) writeMessage(m proto.Message) error {
	data, err := proto.Marshal(m)
	if err != nil {
		return err
	}

	// EncodeRawBytes includes the varint length.
	buf := proto.NewBuffer(make([]byte, 0))
	buf.EncodeRawBytes(data)

	_, err = rc.c.Write(buf.Bytes())
	if err != nil {
		return err
	}
	return nil
}

func (rc *rpcContext) readMessage(m proto.Message) error {
	data := make([]byte, 1024)
	length, err := rc.c.Read(data)
	if err != nil {
		return err
	}

	sliced := data[0:length]
	buf := proto.NewBuffer(sliced)
	_, err = buf.DecodeVarint()
	if err != nil {
		return err
	}

	err = buf.Unmarshal(m)
	if err != nil {
		return err
	}

	return nil
}

type rpcDispatcher struct {
	handlers map[pb.QueryType]rpcHandler
}

func newRpcDispatcher() *rpcDispatcher {
	handlers := make(map[pb.QueryType]rpcHandler)
	return &rpcDispatcher{
		handlers: handlers,
	}
}

func (rd *rpcDispatcher) AddHandler(qt pb.QueryType, handler func(*rpcContext, *pb.WireMessageQuery) error) {
	rd.handlers[qt] = handler
}

func (rd *rpcDispatcher) handleRequest(c net.Conn) {
	defer c.Close()

	rc := &rpcContext{
		c: c,
	}
	wireQuery := &pb.WireMessageQuery{}
	err := rc.readMessage(wireQuery)
	if err != nil {
		rc.writeError("Error reading message.")
		log.Printf("Error reading: %v", err.Error())
		return
	}

	log.Printf("Header: %v", wireQuery.Type)

	handler := rd.handlers[wireQuery.Type]
	if handler == nil {
		rc.writeError("Unknown message.")
		log.Printf("Error handling RPC %v", "No handler")
		return
	}
	err = handler(rc, wireQuery)
	if err != nil {
		rc.writeError("Error handling message.")
		log.Printf("Error handling RPC %v", err.Error())
		return
	}

	log.Printf("Done")
}
