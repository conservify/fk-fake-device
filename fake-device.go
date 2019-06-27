package main

import (
	"crypto/sha1"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strings"

	"github.com/grandcat/zeroconf"

	"github.com/golang/protobuf/proto"

	pb "github.com/fieldkit/app-protocol"
)

func publishAddressOverZeroConf(name string, port int) *zeroconf.Server {
	serviceType := "_fk._tcp"

	server, err := zeroconf.Register(name, serviceType, "local.", port, []string{"txtv=0", "lo=1", "la=2"}, nil)
	if err != nil {
		panic(err)
	}

	log.Printf("Registered ZeroConf: %v %v", name, serviceType)

	return server
}

func writeFile(fn string, msg proto.Message) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	buf := proto.NewBuffer(make([]byte, 0))
	buf.EncodeRawBytes(data)

	err = ioutil.WriteFile(fn, buf.Bytes(), 0644)
	if err != nil {
		return err
	}

	log.Printf("Wrote %s...", fn)

	return nil
}

func writeQueries() {
	writeFile("query-caps.bin", &pb.WireMessageQuery{
		Type: pb.QueryType_QUERY_CAPABILITIES,
	})
	writeFile("query-status.bin", &pb.WireMessageQuery{
		Type: pb.QueryType_QUERY_STATUS,
	})
	writeFile("query-files.bin", &pb.WireMessageQuery{
		Type: pb.QueryType_QUERY_FILES,
	})
	writeFile("query-download-file.bin", &pb.WireMessageQuery{
		Type: pb.QueryType_QUERY_DOWNLOAD_FILE,
	})
	writeFile("query-rename.bin", &pb.WireMessageQuery{
		Type: pb.QueryType_QUERY_CONFIGURE_IDENTITY,
		Identity: &pb.Identity{
			Device: "My Fancy Station",
			Stream: "",
		},
	})
}

type Options struct {
	WriteQueries bool
	Names        string
}

type HardwareState struct {
	Identity pb.Identity
}

type FakeDevice struct {
	Name      string
	Port      int
	ZeroConf  *zeroconf.Server
	WebServer *httpServer
	State     *HardwareState
}

func (fd *FakeDevice) Start(dispatcher *dispatcher) {
	fd.ZeroConf = publishAddressOverZeroConf(fd.Name, fd.Port)

	ws, err := newHttpServer(fd, dispatcher)
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

	flag.BoolVar(&o.WriteQueries, "write-queries", false, "")
	flag.StringVar(&o.Names, "names", "fake0", "")

	flag.Parse()

	if o.WriteQueries {
		log.Printf("Writing sample query files...")

		writeQueries()
	}

	names := strings.Split(o.Names, ",")
	devices := CreateFakeDevicesNamed(names)

	dispatcher := newDispatcher()
	dispatcher.AddHandler(pb.QueryType_QUERY_CAPABILITIES, handleQueryCapabilities)
	dispatcher.AddHandler(pb.QueryType_QUERY_STATUS, handleQueryStatus)
	dispatcher.AddHandler(pb.QueryType_QUERY_FILES, handleQueryFiles)
	dispatcher.AddHandler(pb.QueryType_QUERY_DOWNLOAD_FILE, handleDownloadFile)
	dispatcher.AddHandler(pb.QueryType_QUERY_CONFIGURE_IDENTITY, handleConfigureIdentity)
	dispatcher.AddHandler(pb.QueryType_QUERY_IDENTITY, handleQueryIdentity)

	ts, err := newTcpServer(dispatcher)
	if err != nil {
		panic(err)
	}
	defer ts.Close()

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
