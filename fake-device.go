package main

import (
	"crypto/sha1"
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"

	"github.com/grandcat/zeroconf"

	pb "github.com/fieldkit/app-protocol"
)

func PublishAddressOverZeroConf(name string, port int) *zeroconf.Server {
	serviceType := "_fk._tcp"

	server, err := zeroconf.Register(name, serviceType, "local.", port, []string{"txtv=0", "lo=1", "la=2"}, nil)
	if err != nil {
		panic(err)
	}

	log.Printf("Registered ZeroConf: %v %v", name, serviceType)

	return server
}

type Options struct {
	Names string
}

type StreamState struct {
	Time    uint64
	Size    uint64
	Version uint32
	Record  uint64
}

type HardwareState struct {
	Identity pb.Identity
	Streams  [2]StreamState
}

type FakeDevice struct {
	Name      string
	Port      int
	ZeroConf  *zeroconf.Server
	WebServer *HttpServer
	State     *HardwareState
}

func (fd *FakeDevice) Start(dispatcher *Dispatcher) {
	fd.ZeroConf = PublishAddressOverZeroConf(fd.Name, fd.Port)

	ws, err := NewHttpServer(fd, dispatcher)
	if err != nil {
		panic(err)
	}

	fd.WebServer = ws
}

func (fd *FakeDevice) Close() {
	fd.ZeroConf.Shutdown()
	fd.WebServer.Close()
}

func CreateFakeDevicesNamed(names []string) []*FakeDevice {
	devices := make([]*FakeDevice, len(names))
	for i, name := range names {
		hasher := sha1.New()
		hasher.Write([]byte(name))
		deviceID := hasher.Sum(nil)

		state := HardwareState{
			Identity: pb.Identity{
				DeviceId: deviceID,
				Device:   name,
				Stream:   "",
				Firmware: "91150ca5b2b09608058da273e1181d02cabb2d53",
				Build:    "fk-bundled-fkb.elf_JACOB-WORK_20190809_214014",
			},
			Streams: [2]StreamState{
				StreamState{
					Time:    0,
					Size:    0,
					Version: 0,
					Record:  0,
				},
				StreamState{
					Time:    0,
					Size:    0,
					Version: 0,
					Record:  0,
				},
			},
		}

		devices[i] = &FakeDevice{
			Name:  name,
			Port:  2380 + i,
			State: &state,
		}
	}
	return devices
}

func main() {
	o := Options{}

	flag.StringVar(&o.Names, "names", "fake0", "")
	flag.Parse()

	names := strings.Split(o.Names, ",")
	devices := CreateFakeDevicesNamed(names)

	dispatcher := NewDispatcher()
	dispatcher.AddHandler(pb.QueryType_QUERY_STATUS, handleQueryStatus)

	for _, device := range devices {
		device.Start(dispatcher)
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
