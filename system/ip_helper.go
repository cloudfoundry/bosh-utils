package system

import (
	"encoding/binary"
	"fmt"
	"net"
)

func CalculateNetworkAndBroadcast(ipAddress, netmask string) (string, string, error) {
	ip := net.ParseIP(ipAddress)
	if ip == nil {
		return "", "", fmt.Errorf("Invalid IP '%s'", ipAddress)
	}

	mask := net.ParseIP(netmask)
	if mask == nil {
		return "", "", fmt.Errorf("Invalid netmask '%s'", netmask)
	}

	ip = ip.To4()
	mask = mask.To4()

	if ip != nil && mask != nil {
		return calculateV4NetworkAndBroadcast(ip, mask)
	}

	return "", "", nil
}

func calculateV4NetworkAndBroadcast(ipAddress, netmask net.IP) (string, string, error) {
	mask := net.IPMask(netmask)
	broadcast := make(net.IP, net.IPv4len)

	binary.BigEndian.PutUint32(broadcast,
		binary.BigEndian.Uint32(ipAddress.To4())|^binary.BigEndian.Uint32(netmask.To4()))

	network := ipAddress.Mask(mask)
	if network == nil {
		return "", "", fmt.Errorf("could not apply mask %v to IP address %v", mask, ipAddress)
	}

	return network.To4().String(), broadcast.To4().String(), nil
}
