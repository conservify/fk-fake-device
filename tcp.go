package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/golang/protobuf/proto"

	pb "github.com/fieldkit/app-protocol"
)

const (
	PORT = 12345
)

type tcpReplyWriter struct {
	c net.Conn
}

type tcpServer struct {
	dispatcher *dispatcher
	listener   net.Listener
}

func newTcpServer(dispatcher *dispatcher) (*tcpServer, error) {
	log.Printf("(tcp) Listening on %d", PORT)

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", PORT))
	if err != nil {
		return nil, err
	}

	ts := &tcpServer{
		dispatcher: dispatcher,
		listener:   l,
	}

	go func() {
		ctx := context.Background()

		for {
			conn, err := l.Accept()
			if err != nil {
				log.Printf("Error accepting: " + err.Error())
				time.Sleep(1 * time.Second)
			}

			log.Printf("New connection...")

			ts.handle(ctx, conn)
		}
	}()

	return ts, nil
}

func (ts *tcpServer) handle(ctx context.Context, c net.Conn) {
	defer c.Close()

	rc := &rpcReplyWriter{
		c: c,
	}

	wireQuery := &pb.WireMessageQuery{}
	err := rc.ReadMessage(wireQuery)
	if err != nil {
		rc.WriteError("Error reading message.")
		log.Printf("Error reading: %v", err.Error())
		return
	}

	log.Printf("Header: %v", wireQuery.Type)

	handler := ts.dispatcher.handlers[wireQuery.Type]
	if handler == nil {
		rc.WriteError("Unknown message.")
		log.Printf("Error handling RPC %v", "No handler")
		return
	}

	err = handler(ctx, rc)
	if err != nil {
		rc.WriteError("Error handling message.")
		log.Printf("Error handling RPC %v", err.Error())
		return
	}

	log.Printf("Done")
}

func (ts *tcpServer) Close() {
	ts.listener.Close()
}

type rpcReplyWriter struct {
	size int
	c    net.Conn
}

func (rc *rpcReplyWriter) ReadMessage(m proto.Message) error {
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

func (rw *rpcReplyWriter) Prepare(size int) error {
	rw.size = size

	return nil
}

func (rw *rpcReplyWriter) WriteReply(m *pb.WireMessageReply) (int, error) {
	data, err := proto.Marshal(m)
	if err != nil {
		return 0, err
	}

	buf := proto.NewBuffer(make([]byte, 0))

	buf.EncodeRawBytes(data)

	return rw.WriteBytes(buf.Bytes())
}

func (rw *rpcReplyWriter) WriteBytes(bytes []byte) (int, error) {
	return rw.c.Write(bytes)
}

func (rw *rpcReplyWriter) WriteError(message string) (int, error) {
	wireReply := &pb.WireMessageReply{
		Type: pb.ReplyType_REPLY_ERROR,
		Errors: []*pb.Error{
			&pb.Error{
				Message: message,
			},
		},
	}

	return rw.WriteReply(wireReply)
}
