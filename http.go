package main

import (
	"context"
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
		buf := proto.NewBuffer(bytes)
		wireQuery := &pb.WireMessageQuery{}
		err = buf.Unmarshal(wireQuery)
		if err != nil {
			return nil, err
		}

		log.Printf("(http) Query: %v", wireQuery)

		handler := hs.dispatcher.handlers[wireQuery.Type]
		if handler == nil {
			hs.writeError(res, "Unknown message.")
			log.Printf("Error handling RPC %v", "No handler")
			return
		}

		reply, err := handler(ctx, wireQuery)
		if err != nil {
			hs.writeError(res, "Error handling message.")
			log.Printf("Error handling RPC %v", err.Error())
			return
		}

		hs.writeMessage(res, reply)

		return nil, nil
	})
	if err != nil {
		panic(err)
	}
}

func (hs *httpServer) Close() {
}

func (hs *httpServer) writeMessage(res http.ResponseWriter, m proto.Message) error {
	data, err := proto.Marshal(m)
	if err != nil {
		return err
	}

	buf := proto.NewBuffer(make([]byte, 0))
	buf.EncodeRawBytes(data)

	res.Header().Set("Content-Type", "application/vnd.fk.data+binary")

	_, err = res.Write(buf.Bytes())
	if err != nil {
		return err
	}

	log.Printf("(http) Wrote %d byte reply", len(buf.Bytes()))

	return nil
}

func (hs *httpServer) writeError(res http.ResponseWriter, message string) error {
	wireReply := &pb.WireMessageReply{
		Type: pb.ReplyType_REPLY_ERROR,
		Errors: []*pb.Error{
			&pb.Error{
				Message: message,
			},
		},
	}

	return hs.writeMessage(res, wireReply)
}
