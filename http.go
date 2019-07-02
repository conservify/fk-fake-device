package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/golang/protobuf/proto"

	pb "github.com/fieldkit/app-protocol"
)

type httpServer struct {
	dispatcher *dispatcher
	device     *FakeDevice
}

func newHttpServer(device *FakeDevice, dispatcher *dispatcher) (*httpServer, error) {
	hs := &httpServer{
		dispatcher: dispatcher,
		device:     device,
	}

	notFoundHandler := http.NotFoundHandler()

	server := http.NewServeMux()
	server.Handle("/fk/v1", hs)
	server.HandleFunc("/fk/v1/download", func(w http.ResponseWriter, req *http.Request) {
		junk := make([]byte, 1024)
		size := 1024 * 1024
		bytes := 0
		desired := req.URL.Query()["size"]

		if len(desired) == 1 {
			i, err := strconv.Atoi(desired[0])
			if err != nil {
				panic(err)
			}
			size = i * len(junk)
		}

		log.Printf("(http) serving junk data (%v bytes)", size)

		rw := &httpReplyWriter{
			hexEncoding: false,
			res:         w,
		}

		rw.Prepare(size)
		rw.WriteHeaders()
		for bytes < size {
			rw.WriteBytes(junk)
			bytes += len(junk)
		}
	})
	server.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		log.Printf("Unknown URL: %s", req.URL)
		notFoundHandler.ServeHTTP(w, req)
	})

	sslPort := device.Port + 1000

	go http.ListenAndServe(fmt.Sprintf(":%d", device.Port), server)
	log.Printf("(http) Listening on %d", device.Port)

	go http.ListenAndServeTLS(fmt.Sprintf(":%d", sslPort), "server_dev.crt", "server_dev.key", server)
	log.Printf("(https) Listening on %d", sslPort)

	return hs, nil
}

func (hs *httpServer) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	ctx := context.Background()

	log.Printf("(http) Request: %v %v", req.RemoteAddr, req)

	contentType := req.Header.Get("Content-Type")

	var reader io.Reader = req.Body

	/* Hack to support hex encoded encoding. */
	hexEncoding := contentType == "text/plain"
	if hexEncoding {
		reader = hex.NewDecoder(req.Body)
	}

	_, _, err := ReadLengthPrefixedCollection(ctx, MaximumDataRecordLength, reader, func(bytes []byte) (m proto.Message, err error) {
		rw := &httpReplyWriter{
			hexEncoding: hexEncoding,
			res:         res,
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

		err = handler(ctx, hs.device, wireQuery, rw)
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
	hexEncoding bool
	headers     bool
	size        int
	res         http.ResponseWriter
}

func (rw *httpReplyWriter) WriteHeaders() error {
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

	if rw.hexEncoding {
		rw.size += hex.EncodedLen(len(bytes)) /* This is just N * 2 */
	} else {
		rw.size += len(bytes)
	}

	rw.WriteHeaders()

	return rw.WriteBytes(bytes)
}

func (rw *httpReplyWriter) WriteBytes(bytes []byte) (int, error) {
	if rw.hexEncoding {
		writer := hex.NewEncoder(rw.res)
		return writer.Write(bytes)
	}
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
