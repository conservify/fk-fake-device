package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	pb "github.com/conservify/fk-app-protocol"
	"github.com/golang/protobuf/proto"
	"log"
	"net"
	"os/exec"
	"strconv"
	"time"
)

const (
	PORT = 12345
)

type rpcContext struct {
	c net.Conn
}

type rpcHandler func(rc *rpcContext) error

func (rc *rpcContext) writeMessage(m proto.Message) error {
	data, err := proto.Marshal(m)
	if err != nil {
		return err
	}

	_, err = rc.c.Write(data)
	if err != nil {
		return err
	}
	return nil
}

func (rc *rpcContext) readMessage(m proto.Message) error {
	data := make([]byte, 1024)
	length, err := rc.c.Read(data)
	if err != nil {
		return err
	}

	sliced := data[0:length]
	err = proto.Unmarshal(sliced, m)
	if err != nil {
		return err
	}

	return nil
}

type rpcDispatcher struct {
	handlers map[pb.RequestHeader_MessageType]rpcHandler
}

func rpcPing(rc *rpcContext) error {
	request := &pb.PingRequest{}
	err := rc.readMessage(request)
	if err != nil {
		return err
	}

	log.Printf("Handling %v", *request)

	response := &pb.PingResponse{
		Time: request.Time,
	}
	rc.writeMessage(response)

	return nil
}

func newRpcDispatcher() *rpcDispatcher {
	handlers := make(map[pb.RequestHeader_MessageType]rpcHandler)
	handlers[pb.RequestHeader_PING] = rpcPing
	return &rpcDispatcher{
		handlers: handlers,
	}
}

func (rd *rpcDispatcher) handleRequest(c net.Conn) {
	defer c.Close()

	rc := &rpcContext{
		c: c,
	}
	requestHeader := &pb.RequestHeader{}
	err := rc.readMessage(requestHeader)
	if err != nil {
		log.Printf("Error reading:", err.Error())
		return
	}

	log.Printf("Header: %v", requestHeader.Type)

	handler := rd.handlers[requestHeader.Type]
	err = handler(rc)
	if err != nil {
		log.Printf("Error handling RPC %v", err.Error())
		return
	}

	log.Printf("Done")
}

func getLanIp() (lanIp *net.IP, lanNet *net.IPNet, err error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return
	}
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			return nil, nil, err
		}
		for _, addr := range addrs {
			var ip *net.IP
			var ipNet *net.IPNet

			switch v := addr.(type) {
			case *net.IPNet:
				ip = &v.IP
				ipNet = v
				// case *net.IPAddr:
				// ip = v.IP
			}

			if !ip.IsLoopback() {
				if ip.To4() != nil {
					lanIp = ip
					lanNet = ipNet
					return lanIp, lanNet, nil
				}
			}
		}
	}

	return
}

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

func lastAddr(n *net.IPNet) (net.IP, error) {
	if n.IP.To4() == nil {
		return net.IP{}, errors.New("IPv6 unsupported")
	}
	ip := make(net.IP, len(n.IP.To4()))
	binary.BigEndian.PutUint32(ip, binary.BigEndian.Uint32(n.IP.To4())|^binary.BigEndian.Uint32(net.IP(n.Mask).To4()))
	return ip, nil
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
		fmt.Printf(".")

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
