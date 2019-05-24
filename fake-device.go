package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os/exec"
	"strconv"
	"time"

	"github.com/golang/protobuf/proto"

	pb "github.com/fieldkit/app-protocol"
)

func publishAddressOverMdns() {
	lanIp, _, err := getLanIp()
	if err != nil {
		log.Printf("Error finding LAN ip: %v", err)
	} else {
		cmd := []string{
			"avahi-publish-address",
			"-Rv",
			"fk.local",
			lanIp.String(),
		}
		log.Printf("Command: %v", cmd)

		c := exec.Command(cmd[0], cmd[1:]...)
		c.Run()
	}
}

func publishAddressOverUdp() {
	_, lanNet, err := getLanIp()
	if err != nil {
		log.Fatalf("Error %v", err)
	}

	a, err := lastAddr(lanNet)
	if err != nil {
		log.Fatalf("Error %v", err)
	}

	server, err := net.ResolveUDPAddr("udp", a.String()+":12344")
	if err != nil {
		log.Fatalf("Error %v", err)
	}

	local, err := net.ResolveUDPAddr("udp", ":12345")
	if err != nil {
		log.Fatalf("Error %v", err)
	}

	c, err := net.DialUDP("udp", local, server)
	if err != nil {
		log.Fatalf("Error %v", err)
	}

	defer c.Close()

	i := 0

	for {
		if false {
			fmt.Printf(".")
		}

		msg := strconv.Itoa(i)
		buf := []byte(msg)
		_, err = c.Write(buf)
		if err != nil {
			log.Printf("Error %v", err)
		}

		time.Sleep(1 * time.Second)

		i++
	}
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
}

type options struct {
	WriteQueries bool
}

func main() {
	o := options{}

	flag.BoolVar(&o.WriteQueries, "write-queries", false, "")

	flag.Parse()

	if o.WriteQueries {
		log.Printf("Writing sample query files...")

		writeQueries()
	}

	go publishAddressOverMdns()
	go publishAddressOverUdp()

	dispatcher := newDispatcher()
	dispatcher.AddHandler(pb.QueryType_QUERY_CAPABILITIES, handleQueryCapabilities)
	dispatcher.AddHandler(pb.QueryType_QUERY_STATUS, handleQueryStatus)
	dispatcher.AddHandler(pb.QueryType_QUERY_FILES, handleQueryFiles)

	hs, err := newHttpServer(dispatcher)
	if err != nil {
		panic(err)
	}
	defer hs.Close()
	ts, err := newTcpServer(dispatcher)
	if err != nil {
		panic(err)
	}
	defer ts.Close()

	for {
		time.Sleep(1 * time.Second)
	}
}
