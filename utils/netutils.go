package utils

import (
	"net"
	"errors"
)

func GetFirstNotLoopbackIPv4Address() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if (iface.Flags&net.FlagUp == net.FlagUp) && (iface.Flags&net.FlagLoopback != net.FlagLoopback) && (iface.Flags&net.FlagPointToPoint != net.FlagPointToPoint) {
			addrs, err := iface.Addrs()
			if err != nil {
				return "", err
			}
			for _, addr := range addrs {
				if ipAddr, ok := addr.(*net.IPNet); ok {
					ipV4 := ipAddr.IP.To4()
					if ipV4 != nil {
						return ipV4.String(), nil
					}
				}

				if ipAddr, ok := addr.(*net.IPAddr); ok {
					ipV4 := ipAddr.IP.To4()
					if ipV4 != nil {
						return ipV4.String(), nil
					}
				}
			}
		}
	}
	return "", errors.New("can not found not loopback IPv4 address")
}
