package main

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/golang/protobuf/proto"

	pb "github.com/fieldkit/app-protocol"
)

type HttpServer struct {
	dispatcher *Dispatcher
	device     *FakeDevice
}

func GetDownloadQuery(ctx context.Context, req *http.Request) *pb.DownloadQuery {
	/* Hack to support hex encoded encoding. */
	var reader io.Reader = req.Body
	contentType := req.Header.Get("Content-Type")
	hexEncoding := contentType == "text/plain"
	if hexEncoding {
		reader = hex.NewDecoder(req.Body)
	}
	queries, _, err := ReadLengthPrefixedCollection(ctx, MaximumDataRecordLength, reader, func(bytes []byte) (m proto.Message, err error) {
		buf := proto.NewBuffer(bytes)
		downloadQuery := &pb.DownloadQuery{}
		err = buf.Unmarshal(downloadQuery)
		if err != nil {
			return nil, err
		}

		log.Printf("(http) Query: %v", downloadQuery)

		return downloadQuery, io.EOF
	})
	if err != nil {
		panic(err)
	}

	if len(queries) == 0 {
		start_str := req.URL.Query()["start"]
		end_str := req.URL.Query()["end"]
		if len(start_str) == 1 && len(end_str) == 1 {
			start, err := strconv.Atoi(start_str[0])
			if err != nil {
				panic(err)
			}
			end, err := strconv.Atoi(end_str[0])
			if err != nil {
				panic(err)
			}
			return &pb.DownloadQuery{
				Ranges: []*pb.Range{
					&pb.Range{
						Start: uint32(start),
						End:   uint32(end),
					},
				},
			}
		}
		return nil
	}

	return queries[0].(*pb.DownloadQuery)
}

func HandleDownload(ctx context.Context, w http.ResponseWriter, req *http.Request, stream *StreamState) error {
	start := uint64(0)
	end := stream.Record + 1

	query := GetDownloadQuery(ctx, req)
	if query != nil {
		log.Printf("%v", query)
		start = uint64(query.Ranges[0].Start)
		end = uint64(query.Ranges[0].End)
	}

	start_position := stream.PositionOf(start)
	end_position := stream.PositionOf(end)
	length := end_position - start_position

	log.Printf("(http) Downloading (%d -> %d)", start, end)
	log.Printf("(http) Downloading (%d -> %d) %d bytes", start_position, end_position, length)

	w.Header().Add("Fk-Sync", fmt.Sprintf("%d, %d", start, end))

	file, err := stream.OpenFile()
	if err != nil {
		panic(err)
	}

	rw := &HttpReplyWriter{
		hexEncoding: false,
		res:         w,
	}
	rw.Prepare(int(length))

	buffer := make([]byte, 1024)
	for true {
		header := RecordHeader{}
		err := binary.Read(file, binary.BigEndian, &header)
		if err == io.EOF {
			break
		}

		if header.Record >= start && header.Record < end {
			limited := io.LimitReader(file, int64(header.Size))
			nread, err := io.ReadAtLeast(limited, buffer, int(header.Size))
			if err != nil {
				panic(err)
			}

			rw.WriteBytes(buffer[:nread])
		} else {
			file.Seek(int64(header.Size), 1)
		}
	}

	defer file.Close()

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
		ctx := context.Background()
		HandleDownload(ctx, w, req, device.State.Streams[0])
	})
	server.HandleFunc("/fk/v1/download/data", func(w http.ResponseWriter, req *http.Request) {
		ctx := context.Background()
		HandleDownload(ctx, w, req, device.State.Streams[0])
	})
	server.HandleFunc("/fk/v1/download/1", func(w http.ResponseWriter, req *http.Request) {
		ctx := context.Background()
		HandleDownload(ctx, w, req, device.State.Streams[1])
	})
	server.HandleFunc("/fk/v1/download/meta", func(w http.ResponseWriter, req *http.Request) {
		ctx := context.Background()
		HandleDownload(ctx, w, req, device.State.Streams[1])
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

	log.Printf("(http) Request: %v", req.RemoteAddr)

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

	log.Printf("(http) Writing %d bytes", len(bytes))

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
