package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	_ "strconv"

	"github.com/golang/protobuf/proto"

	pb "github.com/fieldkit/app-protocol"
)

type HttpServer struct {
	dispatcher *Dispatcher
	device     *FakeDevice
}

func HandleDownload(w http.ResponseWriter, req *http.Request) error {
	ctx := context.Background()

	start := 0
	finish := 100

	/* Hack to support hex encoded encoding. */
	var reader io.Reader = req.Body
	contentType := req.Header.Get("Content-Type")
	hexEncoding := contentType == "text/plain"
	if hexEncoding {
		reader = hex.NewDecoder(req.Body)
	}
	_, _, err := ReadLengthPrefixedCollection(ctx, MaximumDataRecordLength, reader, func(bytes []byte) (m proto.Message, err error) {
		buf := proto.NewBuffer(bytes)
		downloadQuery := &pb.DownloadQuery{}
		err = buf.Unmarshal(downloadQuery)
		if err != nil {
			return nil, err
		}

		log.Printf("(http) Query: %v", downloadQuery)

		return nil, io.EOF
	})
	if err != nil {
		panic(err)
	}

	w.Header().Add("Fk-Sync", fmt.Sprintf("%d, %d", start, finish))

	body := proto.NewBuffer(make([]byte, 0))

	for reading := start; reading < finish; reading += 1 {
		record := generateFakeReading(uint32(reading))
		body.EncodeMessage(record)
	}

	size := len(body.Bytes())

	log.Printf("(http) Downloading (%d -> %d) %d bytes", start, finish, size)

	rw := &HttpReplyWriter{
		hexEncoding: false,
		res:         w,
	}

	rw.Prepare(size)
	rw.WriteBytes(body.Bytes())

	return nil
}

func NewHttpServer(device *FakeDevice, dispatcher *Dispatcher) (*HttpServer, error) {
	hs := &HttpServer{
		dispatcher: dispatcher,
		device:     device,
	}

	notFoundHandler := http.NotFoundHandler()

	server := http.NewServeMux()
	server.Handle("/fk/v1", hs)
	server.HandleFunc("/fk/v1/download/0", func(w http.ResponseWriter, req *http.Request) {
		HandleDownload(w, req)
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

func (hs *HttpServer) ServeHTTP(res http.ResponseWriter, req *http.Request) {
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
		rw := &HttpReplyWriter{
			hexEncoding: hexEncoding,
			res:         res,
		}

		buf := proto.NewBuffer(bytes)
		wireQuery := &pb.HttpQuery{}
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

func (hs *HttpServer) Close() {
}

type HttpReplyWriter struct {
	hexEncoding bool
	headers     bool
	size        int
	res         http.ResponseWriter
}

func (rw *HttpReplyWriter) WriteHeaders() error {
	if !rw.headers {
		rw.res.Header().Set("Content-Type", "application/vnd.fk.data+binary")
		rw.res.Header().Set("Content-Length", fmt.Sprintf("%d", rw.size))
		rw.headers = true
	}

	return nil
}

func (rw *HttpReplyWriter) Prepare(size int) error {
	rw.size = size

	return nil
}

func (rw *HttpReplyWriter) WriteReply(m *pb.HttpReply) (int, error) {
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

func (rw *HttpReplyWriter) WriteBytes(bytes []byte) (int, error) {
	rw.WriteHeaders()

	if rw.hexEncoding {
		writer := hex.NewEncoder(rw.res)
		return writer.Write(bytes)
	}
	return rw.res.Write(bytes)
}

func (rw *HttpReplyWriter) WriteError(message string) (int, error) {
	wireReply := &pb.HttpReply{
		Type: pb.ReplyType_REPLY_ERROR,
		Errors: []*pb.Error{
			&pb.Error{
				Message: message,
			},
		},
	}

	return rw.WriteReply(wireReply)
}
