package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/grandcat/zeroconf"

	"github.com/golang/protobuf/proto"

	pb "github.com/fieldkit/app-protocol"
	pbatlas "github.com/fieldkit/atlas-protocol"
)

func PublishAddressOverZeroConf(name string, deviceId string, port int) *zeroconf.Server {
	serviceType := "_fk._tcp"

	server, err := zeroconf.Register(deviceId, serviceType, "local.", port, nil, nil)
	if err != nil {
		panic(err)
	}

	log.Printf("Registered ZeroConf: %v %v %v", name, serviceType, deviceId)

	return server
}

type Options struct {
	Names         string
	NoModules     bool
	PrimeReadings int
}

type StreamState struct {
	Time    uint64
	Size    uint64
	Version uint32
	Record  uint64
	File    string
}

type RecordHeader struct {
	Size   uint32
	Record uint64
}

func (ss *StreamState) Append(body []byte) {
	file, err := os.OpenFile(ss.File, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}

	defer file.Close()

	header := RecordHeader{
		Size:   uint32(len(body)),
		Record: ss.Record,
	}

	var record bytes.Buffer

	binary.Write(&record, binary.BigEndian, header)

	file.Write(record.Bytes())
	file.Write(body)

	ss.Record += 1
	ss.Time = uint64(time.Now().Unix())
	ss.Size += uint64(len(body))
}

func (ss *StreamState) AppendConfiguration() {
	record := generateFakeConfiguration()
	body := proto.NewBuffer(make([]byte, 0))
	body.EncodeMessage(record)
	ss.Append(body.Bytes())
}

func (ss *StreamState) AppendReading() {
	record := generateFakeReading(uint32(ss.Record))
	body := proto.NewBuffer(make([]byte, 0))
	body.EncodeMessage(record)
	ss.Append(body.Bytes())
}

func (ss *StreamState) OpenFile() (*os.File, error) {
	return os.OpenFile(ss.File, os.O_CREATE, 0644)
}

func (ss *StreamState) PositionOf(record uint64) int64 {
	file, err := os.OpenFile(ss.File, os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}

	defer file.Close()

	position := int64(0)

	for true {
		header := RecordHeader{}
		err := binary.Read(file, binary.BigEndian, &header)
		if err == io.EOF {
			break
		}

		if header.Record == record {
			log.Printf("Position(%d) = %d", record, position)

			return position
		}

		_, err = file.Seek(int64(header.Size), 1)
		if err != nil {
			panic(err)
		}

		position += int64(header.Size)
	}

	log.Printf("Position(%d) = %d (EOF)", record, position)

	return position
}

func (ss *StreamState) Open() {
	file, err := os.OpenFile(ss.File, os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}

	defer file.Close()

	for true {
		header := RecordHeader{}
		err := binary.Read(file, binary.BigEndian, &header)
		if err == io.EOF {
			break
		}

		_, err = file.Seek(int64(header.Size), 1)
		if err != nil {
			panic(err)
		}

		ss.Record = header.Record + 1
		ss.Size += uint64(header.Size)
	}

	log.Printf("Opened %s (#%d) (%d bytes)", ss.File, ss.Record, ss.Size)
}

type HardwareState struct {
	Identity      pb.Identity
	Lora          *pb.LoraSettings
	Streams       [2]*StreamState
	Networks      []*pb.NetworkInfo
	ReadingsReady bool
	Recording     bool
	StartedTime   uint64
}

type FakeModule struct {
	SensorType  pbatlas.SensorType
	Calibration uint32
}

type FakeDevice struct {
	Name             string
	DeviceId         string
	Port             int
	ZeroConf         *zeroconf.Server
	WebServer        *HttpServer
	State            *HardwareState
	Latitude         float32
	Longitude        float32
	Modules          []*FakeModule
	ReadingsSchedule *pb.Schedule
}

func (fd *FakeDevice) Start(dispatcher *Dispatcher) {
	fd.ZeroConf = PublishAddressOverZeroConf(fd.Name, fd.DeviceId, fd.Port)

	ws, err := NewHttpServer(fd, dispatcher)
	if err != nil {
		panic(err)
	}

	fd.WebServer = ws
}

func (fd *FakeDevice) Close() {
	log.Printf("%s Close\n", fd.Name)
	fd.ZeroConf.Shutdown()
	fd.WebServer.Close()
}

func (fd *FakeDevice) FakeReadings() {
	fd.State.Streams[0].Open()
	fd.State.Streams[1].Open()

	fd.State.Streams[1].AppendConfiguration()

	for {
		fd.State.Streams[0].AppendReading()

		time.Sleep(5 * time.Second)
	}
}

func CreateFakeDevicesNamed(names []string, noModules bool) []*FakeDevice {
	devices := make([]*FakeDevice, len(names))
	for i, name := range names {
		deviceIdHasher := sha1.New()
		deviceIdHasher.Write([]byte(fmt.Sprintf("station-%s", name)))
		deviceID := deviceIdHasher.Sum(nil)

		generationHasher := sha1.New()
		generationHasher.Write([]byte(fmt.Sprintf("station-%s-generation", name)))
		generation := generationHasher.Sum(nil)

		state := HardwareState{
			Recording:   false,
			StartedTime: 0, // uint64(time.Now().Unix() - 300),
			Lora: &pb.LoraSettings{
				DeviceEui: deviceID,
			},
			Identity: pb.Identity{
				DeviceId:     deviceID,
				GenerationId: generation,
				Device:       name,
				Name:         name,
				Stream:       "",
				Firmware:     "91150ca5b2b09608058da273e1181d02cabb2d53",
				Build:        "fk-bundled-fkb.elf_JACOB-WORK_20190809_214014",
			},
			Networks: []*pb.NetworkInfo{
				&pb.NetworkInfo{
					Ssid:     "Fake",
					Password: "Network",
				},
			},
			Streams: [2]*StreamState{
				&StreamState{
					Time:    0,
					Size:    0,
					Version: 0,
					Record:  0,
					File:    fmt.Sprintf("%s-data.fkpb", name),
				},
				&StreamState{
					Time:    0,
					Size:    0,
					Version: 0,
					Record:  0,
					File:    fmt.Sprintf("%s-meta.fkpb", name),
				},
			},
		}

		devices[i] = &FakeDevice{
			Name:     name,
			DeviceId: hex.EncodeToString(deviceID),
			Port:     2380 + i,
			State:    &state,
			ReadingsSchedule: &pb.Schedule{
				Interval: 60,
				Intervals: []*pb.Interval{
					&pb.Interval{
						Start:    0,
						End:      86400,
						Interval: 60,
					},
				},
			},
			Modules: []*FakeModule{
				&FakeModule{
					SensorType:  pbatlas.SensorType_SENSOR_PH,
					Calibration: 0,
				},
				&FakeModule{
					SensorType:  pbatlas.SensorType_SENSOR_EC,
					Calibration: 0,
				},
				&FakeModule{
					SensorType:  pbatlas.SensorType_SENSOR_TEMP,
					Calibration: 0,
				},
				&FakeModule{
					SensorType:  pbatlas.SensorType_SENSOR_DO,
					Calibration: 0,
				},
				&FakeModule{
					SensorType:  pbatlas.SensorType_SENSOR_ORP,
					Calibration: 0,
				},
			},
		}

		if noModules {
			devices[i].Modules = make([]*FakeModule, 0)
		}
	}
	return devices
}

func main() {
	o := Options{}

	flag.StringVar(&o.Names, "names", "fake0", "")
	flag.BoolVar(&o.NoModules, "no-modules", false, "")
	flag.IntVar(&o.PrimeReadings, "prime-readings", 0, "")
	flag.Parse()

	names := strings.Split(o.Names, ",")
	devices := CreateFakeDevicesNamed(names, o.NoModules)

	if o.PrimeReadings > 0 {
		for _, device := range devices {
			device.State.Streams[0].Open()
			device.State.Streams[1].Open()

			device.State.Streams[1].AppendConfiguration()

			for i := 0; i < o.PrimeReadings; i += 1 {
				device.State.Streams[0].AppendReading()
			}
		}
	}

	dispatcher := NewDispatcher()
	dispatcher.AddHandler(pb.QueryType_QUERY_STATUS, handleQueryStatus)
	dispatcher.AddHandler(pb.QueryType_QUERY_TAKE_READINGS, handleQueryTakeReadings)
	dispatcher.AddHandler(pb.QueryType_QUERY_GET_READINGS, handleQueryReadings)
	dispatcher.AddHandler(pb.QueryType_QUERY_CONFIGURE, handleConfigure)
	dispatcher.AddHandler(pb.QueryType_QUERY_RECORDING_CONTROL, handleRecordingControl)
	dispatcher.AddHandler(pb.QueryType_QUERY_SCAN_NETWORKS, handleQueryScanNetworks)

	for _, device := range devices {
		device.Start(dispatcher)
		go device.FakeReadings()
		defer device.Close()
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	for sig := range c {
		if sig == os.Interrupt {
			break
		}
	}

	log.Printf("Stopped")
}
