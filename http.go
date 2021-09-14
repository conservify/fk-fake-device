package main

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/golang/protobuf/proto"

	"github.com/efarrer/iothrottler"

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

	start := 0
	end := 0

	if len(queries) == 0 {
		firstStr := req.URL.Query()["first"]
		if len(firstStr) == 1 {
			first, err := strconv.Atoi(firstStr[0])
			if err != nil {
				panic(err)
			}

			start = first
		}

		lastStr := req.URL.Query()["last"]
		if len(lastStr) == 1 {
			last, err := strconv.Atoi(lastStr[0])
			if err != nil {
				panic(err)
			}

			end = last
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

	return queries[0].(*pb.DownloadQuery)
}

func HandleDownload(ctx context.Context, w http.ResponseWriter, req *http.Request, device *FakeDevice, stream *StreamState) error {
	start := uint64(0)
	end := stream.Record + 1

	pool := iothrottler.NewIOThrottlerPool(iothrottler.BytesPerSecond * 50 * 1024)

	defer pool.ReleasePool()

	query := GetDownloadQuery(ctx, req)
	if query != nil {
		start = uint64(query.Ranges[0].Start)
		end = uint64(query.Ranges[0].End)
	}

	startPosition := stream.PositionOf(start)
	endPosition := stream.PositionOf(end)
	length := endPosition - startPosition
	headOnly := req.Method == "HEAD"

	log.Printf("(http) Downloading (%d -> %d)", start, end)
	log.Printf("(http) Downloading (%d -> %d) %d bytes", startPosition, endPosition, length)

	w.Header().Add("Fk-Blocks", fmt.Sprintf("%d, %d", start, end))
	w.Header().Add("Fk-Generation", fmt.Sprintf("%s", hex.EncodeToString(device.State.Identity.GenerationId)))
	w.Header().Add("Fk-DeviceId", fmt.Sprintf("%s", hex.EncodeToString(device.State.Identity.DeviceId)))

	file, err := stream.OpenFile()
	if err != nil {
		return nil
	}

	rw := &HttpReplyWriter{
		hexEncoding: false,
		res:         w,
	}

	rw.Prepare(int(length))

	if headOnly {
		rw.WriteHeaders(204)
		return nil
	}

	rw.WriteHeaders(200)

	if err := rw.Throttle(pool); err != nil {
		return nil
	}

	bytesWritten := 0
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
				return nil
			}

			rw.WriteBytes(buffer[:nread])

			bytesWritten += nread
		} else {
			file.Seek(int64(header.Size), 1)
		}
	}

	defer file.Close()

	return nil
}

func HandleFirmware(ctx context.Context, res http.ResponseWriter, req *http.Request, device *FakeDevice) error {
	log.Printf("(http) Request: %v %v", req.RemoteAddr, req.Method)

	contentType := req.Header.Get("Content-Type")

	/* Hack to support hex encoded encoding. */
	hexEncoding := contentType == "text/plain"

	rw := &HttpReplyWriter{
		hexEncoding: hexEncoding,
		res:         res,
	}

	io.Copy(ioutil.Discard, req.Body)

	if false {
		_, err := rw.WriteBytes([]byte{})
		if err != nil {
			return err
		}
	} else {
		_, err := rw.WriteStatusBytes(500, []byte("{ \"success\": true }"))
		if err != nil {
			return err
		}
	}

	return nil
}

func HandleModule(ctx context.Context, res http.ResponseWriter, req *http.Request, device *FakeDevice, position int) error {
	log.Printf("(http) Request: %v %v", req.RemoteAddr, req.Method)

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
		wireQuery := &pb.ModuleHttpQuery{}
		err = buf.Unmarshal(wireQuery)
		if err != nil {
			return nil, err
		}

		log.Printf("(http) module-query[%d]: %v", position, wireQuery)

		reply := &pb.ModuleHttpReply{}
		reply.Type = pb.ModuleReplyType_MODULE_REPLY_SUCCESS
		reply.Configuration = wireQuery.Configuration

		data, err := proto.Marshal(reply)
		if err != nil {
			panic(err)
		}
		buf = proto.NewBuffer(make([]byte, 0))
		buf.EncodeRawBytes(data)

		_, err = rw.WriteBytes(buf.Bytes())

		log.Printf("(http) module-reply[%d]: %v", position, len(reply.Configuration))

		for _, m := range device.Modules {
			if m.Position == position {
				m.Configuration = reply.Configuration
			}
		}

		return nil, io.EOF
	})
	if err != nil {
		panic(err)
	}

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
	server.HandleFunc("/fk/v1/download/data", func(w http.ResponseWriter, req *http.Request) {
		ctx := context.Background()
		HandleDownload(ctx, w, req, device, device.State.Streams[0])
	})
	server.HandleFunc("/fk/v1/download/meta", func(w http.ResponseWriter, req *http.Request) {
		ctx := context.Background()
		HandleDownload(ctx, w, req, device, device.State.Streams[1])
	})
	server.HandleFunc("/fk/v1/modules/0", func(w http.ResponseWriter, req *http.Request) {
		ctx := context.Background()
		HandleModule(ctx, w, req, device, 0)
	})
	server.HandleFunc("/fk/v1/modules/1", func(w http.ResponseWriter, req *http.Request) {
		ctx := context.Background()
		HandleModule(ctx, w, req, device, 1)
	})
	server.HandleFunc("/fk/v1/modules/2", func(w http.ResponseWriter, req *http.Request) {
		ctx := context.Background()
		HandleModule(ctx, w, req, device, 2)
	})
	server.HandleFunc("/fk/v1/modules/3", func(w http.ResponseWriter, req *http.Request) {
		ctx := context.Background()
		HandleModule(ctx, w, req, device, 3)
	})
	server.HandleFunc("/fk/v1/modules/4", func(w http.ResponseWriter, req *http.Request) {
		ctx := context.Background()
		HandleModule(ctx, w, req, device, 4)
	})
	server.HandleFunc("/fk/v1/upload/firmware", func(w http.ResponseWriter, req *http.Request) {
		ctx := context.Background()
		HandleFirmware(ctx, w, req, device)
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

	log.Printf("(http) Request: %v %v", req.RemoteAddr, req.Method)

	contentType := req.Header.Get("Content-Type")
	contentLength := req.Header.Get("Content-Length")

	log.Printf("(http) Content: %v %v", contentType, contentLength)

	var reader io.Reader = req.Body

	/* Hack to support hex encoded encoding. */
	hexEncoding := contentType == "text/plain"
	if hexEncoding {
		reader = hex.NewDecoder(req.Body)
	}

	rw := &HttpReplyWriter{
		hexEncoding: hexEncoding,
		res:         res,
	}

	_, i, err := ReadLengthPrefixedCollection(ctx, MaximumDataRecordLength, reader, func(bytes []byte) (m proto.Message, err error) {
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
			log.Printf("Error handling RPC %v (%v)", "No handler", wireQuery.Type)
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

	if i == 0 {
		handler := hs.dispatcher.handlers[pb.QueryType_QUERY_STATUS]
		if handler == nil {
			panic("pb.QueryType_QUERY_STATUS")
		}

		err = handler(ctx, hs.device, nil, rw)
		if err != nil {
			rw.WriteError("Error handling message.")
			log.Printf("Error handling RPC %v", err.Error())
			return
		}
	}
}

func (hs *HttpServer) Close() {
}

type HttpReplyWriter struct {
	hexEncoding bool
	headers     bool
	size        int
	res         http.ResponseWriter
	writer      io.Writer
}

func (rw *HttpReplyWriter) WriteHeaders(statusCode int) error {
	if !rw.headers {
		log.Printf("(http) write headers %v", rw.size)
		if len(rw.res.Header().Get("Content-Length")) == 0 {
			rw.res.Header().Set("Content-Length", fmt.Sprintf("%d", rw.size))
		}
		if len(rw.res.Header().Get("Content-Type")) == 0 {
			rw.res.Header().Set("Content-Type", "application/vnd.fk.data+binary")
		}
		if len(rw.res.Header().Get("Fk-Bytes")) == 0 {
			rw.res.Header().Set("Fk-Bytes", fmt.Sprintf("%d", rw.size))
		}
		if len(rw.res.Header().Get("Fk-Blocks")) == 0 {
			rw.res.Header().Set("Fk-Blocks", fmt.Sprintf("%d,%d", 0, 0))
		}
		rw.res.WriteHeader(statusCode)
		rw.headers = true
	}

	return nil
}

func (rw *HttpReplyWriter) Close() error {
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

	log.Printf("(http) Writing %d bytes", len(bytes))

	return rw.WriteBytes(bytes)
}

func (rw *HttpReplyWriter) WriteStatusBytes(statusCode int, bytes []byte) (int, error) {
	rw.size = len(bytes)
	err := rw.WriteHeaders(statusCode)
	if err != nil {
		return 0, err
	}
	return rw.WriteBytes(bytes)
}

func (rw *HttpReplyWriter) Throttle(pool *iothrottler.IOThrottlerPool) error {
	w, err := pool.AddWriter(rw)
	if err != nil {
		return err
	}
	rw.writer = w
	return nil
}

func (rw *HttpReplyWriter) Write(bytes []byte) (int, error) {
	return rw.res.Write(bytes)
}

func (rw *HttpReplyWriter) WriteBytes(bytes []byte) (int, error) {
	if !rw.headers {
		if rw.hexEncoding {
			rw.size += hex.EncodedLen(len(bytes)) /* This is just N * 2 */
		} else {
			rw.size += len(bytes)
		}
	}

	rw.WriteHeaders(200)

	if rw.writer == nil {
		rw.writer = rw.res
	}

	if rw.hexEncoding {
		writer := hex.NewEncoder(rw.writer)
		return writer.Write(bytes)
	}
	return rw.writer.Write(bytes)
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
