package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/golang/protobuf/proto"

	pb "github.com/fieldkit/app-protocol"
)

type httpServer struct {
	dispatcher *dispatcher
}

func newHttpServer(dispatcher *dispatcher) (*httpServer, error) {
	hs := &httpServer{
		dispatcher: dispatcher,
	}

	http.Handle("/fk/v1", hs)

	go http.ListenAndServe(":2382", nil)

	log.Printf("(http) Listening on 2382")

	return hs, nil
}

func (hs *httpServer) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	ctx := context.Background()

	_, _, err := ReadLengthPrefixedCollection(ctx, MaximumDataRecordLength, req.Body, func(bytes []byte) (m proto.Message, err error) {
		rw := &httpReplyWriter{
			res: res,
		}

		buf := proto.NewBuffer(bytes)
		wireQuery := &pb.WireMessageQuery{}
		err = buf.Unmarshal(wireQuery)
		if err != nil {
			return nil, err
		}

		log.Printf("(http) Query: %v", wireQuery)

		handler := hs.dispatcher.handlers[wireQuery.Type]
		if handler == nil {
			rw.WriteError("Unknown message.")
			log.Printf("Error handling RPC %v", "No handler")
			return
		}

		err = handler(ctx, rw)
		if err != nil {
			rw.WriteError("Error handling message.")
			log.Printf("Error handling RPC %v", err.Error())
			return
		}

		return nil, io.EOF
	})
	if err != nil {
		panic(err)
	}
}

func (hs *httpServer) Close() {
}

type httpReplyWriter struct {
	headers bool
	size    int
	res     http.ResponseWriter
}

func (rw *httpReplyWriter) writeHeaders() error {
	if !rw.headers {
		rw.res.Header().Set("Content-Type", "application/vnd.fk.data+binary")
		rw.res.Header().Set("Content-Length", fmt.Sprintf("%d", rw.size))
		rw.headers = true
	}

	return nil
}

func (rw *httpReplyWriter) Prepare(size int) error {
	rw.size = size

	return nil
}

func (rw *httpReplyWriter) WriteReply(m *pb.WireMessageReply) (int, error) {
	data, err := proto.Marshal(m)
	if err != nil {
		return 0, err
	}

	buf := proto.NewBuffer(make([]byte, 0))

	buf.EncodeRawBytes(data)

	bytes := buf.Bytes()

	rw.size += len(bytes)

	rw.writeHeaders()

	return rw.WriteBytes(bytes)
}

func (rw *httpReplyWriter) WriteBytes(bytes []byte) (int, error) {
	return rw.res.Write(bytes)
}

func (rw *httpReplyWriter) WriteError(message string) (int, error) {
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
