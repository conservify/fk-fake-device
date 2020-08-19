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

	pb "github.com/fieldkit/app-protocol"
	pbatlas "github.com/fieldkit/atlas-protocol"
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
		startStr := req.URL.Query()["first"]
		if len(startStr) == 1 {
			start, err := strconv.Atoi(startStr[0])
			if err != nil {
				panic(err)
			}
			return &pb.DownloadQuery{
				Ranges: []*pb.Range{
					&pb.Range{
						Start: uint32(start),
					},
				},
			}
		}
		return nil
	}

	return queries[0].(*pb.DownloadQuery)
}

func HandleDownload(ctx context.Context, w http.ResponseWriter, req *http.Request, device *FakeDevice, stream *StreamState) error {
	start := uint64(0)
	end := stream.Record + 1

	query := GetDownloadQuery(ctx, req)
	if query != nil {
		start = uint64(query.Ranges[0].Start)
	}

	startPosition := stream.PositionOf(start)
	endPosition := stream.PositionOf(end)
	length := endPosition - startPosition

	log.Printf("(http) Downloading (%d -> %d)", start, end)
	log.Printf("(http) Downloading (%d -> %d) %d bytes", startPosition, endPosition, length)

	w.Header().Add("Fk-Blocks", fmt.Sprintf("%d, %d", start, end))
	w.Header().Add("Fk-Generation", fmt.Sprintf("%s", hex.EncodeToString(device.State.Identity.Generation)))
	w.Header().Add("Fk-DeviceId", fmt.Sprintf("%s", hex.EncodeToString(device.State.Identity.DeviceId)))

	file, err := stream.OpenFile()
	if err != nil {
		panic(err)
	}

	rw := &HttpReplyWriter{
		hexEncoding: false,
		res:         w,
	}
	rw.Prepare(int(length))
	rw.WriteHeaders(200)

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
				panic(err)
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
		_, err := rw.WriteStatusBytes(500, []byte("{ \"sd_card\": true }"))
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
		wireQuery := &pbatlas.WireAtlasQuery{}
		err = buf.Unmarshal(wireQuery)
		if err != nil {
			return nil, err
		}

		log.Printf("(http) Atlas Query: %v", wireQuery)

		if wireQuery.Calibration != nil {
			switch wireQuery.Calibration.Operation {
			case pbatlas.CalibrationOperation_CALIBRATION_CLEAR:
				device.Modules[position].Calibration = 0
				log.Printf("(http) atlas-operation: CLEAR")
			case pbatlas.CalibrationOperation_CALIBRATION_SET:
				which := wireQuery.Calibration.Which
				value := wireQuery.Calibration.Value
				previous := device.Modules[position].Calibration
				switch device.Modules[position].SensorType {
				case pbatlas.SensorType_SENSOR_PH:
					switch pbatlas.PhCalibrateCommand(which) {
					case pbatlas.PhCalibrateCommand_CALIBRATE_PH_LOW:
						device.Modules[position].Calibration |= uint32(pbatlas.PhCalibrations_PH_LOW)
					case pbatlas.PhCalibrateCommand_CALIBRATE_PH_MIDDLE:
						device.Modules[position].Calibration |= uint32(pbatlas.PhCalibrations_PH_MIDDLE)
					case pbatlas.PhCalibrateCommand_CALIBRATE_PH_HIGH:
						device.Modules[position].Calibration |= uint32(pbatlas.PhCalibrations_PH_HIGH)
					default:
						log.Printf("(http) unknown calibration")
					}
				case pbatlas.SensorType_SENSOR_ORP:
					switch pbatlas.OrpCalibrateCommand(which) {
					case pbatlas.OrpCalibrateCommand_CALIBRATE_ORP_SINGLE:
						device.Modules[position].Calibration |= uint32(pbatlas.OrpCalibrations_ORP_SINGLE)
					default:
						log.Printf("(http) unknown calibration")
					}
				case pbatlas.SensorType_SENSOR_DO:
					switch pbatlas.DoCalibrateCommand(which) {
					case pbatlas.DoCalibrateCommand_CALIBRATE_DO_ATMOSPHERE:
						device.Modules[position].Calibration |= uint32(pbatlas.DoCalibrations_DO_ATMOSPHERE)
					case pbatlas.DoCalibrateCommand_CALIBRATE_DO_ZERO:
						device.Modules[position].Calibration |= uint32(pbatlas.DoCalibrations_DO_ZERO)
					default:
						log.Printf("(http) unknown calibration")
					}
				case pbatlas.SensorType_SENSOR_TEMP:
					switch pbatlas.TempCalibrateCommand(which) {
					case pbatlas.TempCalibrateCommand_CALIBRATE_TEMP_SINGLE:
						device.Modules[position].Calibration |= uint32(pbatlas.TempCalibrations_TEMP_SINGLE)
					default:
						log.Printf("(http) unknown calibration")
					}
				case pbatlas.SensorType_SENSOR_EC:
					switch pbatlas.EcCalibrateCommand(which) {
					case pbatlas.EcCalibrateCommand_CALIBRATE_EC_DRY:
						device.Modules[position].Calibration |= uint32(pbatlas.EcCalibrations_EC_DRY)
					case pbatlas.EcCalibrateCommand_CALIBRATE_EC_SINGLE:
						device.Modules[position].Calibration |= uint32(pbatlas.EcCalibrations_EC_SINGLE)
					case pbatlas.EcCalibrateCommand_CALIBRATE_EC_DUAL_LOW:
						device.Modules[position].Calibration |= uint32(pbatlas.EcCalibrations_EC_DUAL_LOW)
					case pbatlas.EcCalibrateCommand_CALIBRATE_EC_DUAL_HIGH:
						device.Modules[position].Calibration |= uint32(pbatlas.EcCalibrations_EC_DUAL_HIGH)
					default:
						log.Printf("(http) unknown calibration")
					}
				default:
					log.Printf("(http) unknown sensor")
				}
				log.Printf("(http) atlas-operation: SET %v %v (%v -> %v)", which, value, previous, device.Modules[position].Calibration)
			}
		}

		reply := generateAtlasStatus(device, position, true)

		_, err = rw.WriteBytes(reply)

		log.Printf("(http) Atlas Reply: %v", len(reply))

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
}

func (hs *HttpServer) Close() {
}

type HttpReplyWriter struct {
	hexEncoding bool
	headers     bool
	size        int
	res         http.ResponseWriter
}

func (rw *HttpReplyWriter) WriteHeaders(statusCode int) error {
	if !rw.headers {
		log.Printf("(http) write headers %v", rw.size)
		rw.res.Header().Set("Content-Type", "application/vnd.fk.data+binary")
		rw.res.Header().Set("Content-Length", fmt.Sprintf("%d", rw.size))
		rw.res.WriteHeader(statusCode)
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

func (rw *HttpReplyWriter) WriteBytes(bytes []byte) (int, error) {
	if !rw.headers {
		if rw.hexEncoding {
			rw.size += hex.EncodedLen(len(bytes)) /* This is just N * 2 */
		} else {
			rw.size += len(bytes)
		}
	}

	rw.WriteHeaders(200)

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
