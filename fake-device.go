package main

import (
	"fmt"
	pb "github.com/conservify/fk-app-protocol"
	"log"
	"net"
	"os/exec"
	"strconv"
	"time"
)

const (
	PORT = 12345
)

func publishAddressOverMdns() {
	lanIp, _, err := getLanIp()
	if err != nil {
		log.Printf("Error finding LAN ip: %v", err)
	} else {
		cmd := []string{
			"avahi-publish-address",
			"-Rv",
			"noaa-ctd.local",
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

func main() {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", PORT))
	if err != nil {
		log.Fatalf("Error listening:" + err.Error())
	}

	defer l.Close()

	go publishAddressOverMdns()

	go publishAddressOverUdp()

	rd := newRpcDispatcher()
	rd.AddHandler(pb.QueryType_QUERY_CAPABILITIES, rpcQueryCapabilities)
	rd.AddHandler(pb.QueryType_QUERY_DATA_SETS, rpcQueryDataSets)

	log.Printf("Listening on %d\n", PORT)
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Printf("Error accepting: " + err.Error())
			time.Sleep(1 * time.Second)
		}

		log.Printf("New connection...")

		go rd.handleRequest(conn)
	}
}
