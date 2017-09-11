package main

import (
	"encoding/binary"
	"errors"
	"net"
)

func lastAddr(n *net.IPNet) (net.IP, error) {
	if n.IP.To4() == nil {
		return net.IP{}, errors.New("IPv6 unsupported")
	}
	ip := make(net.IP, len(n.IP.To4()))
	binary.BigEndian.PutUint32(ip, binary.BigEndian.Uint32(n.IP.To4())|^binary.BigEndian.Uint32(net.IP(n.Mask).To4()))
	return ip, nil
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
