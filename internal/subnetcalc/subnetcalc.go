package subnetcalc

import (
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"strings"
	"text/tabwriter"
)

type SubnetInfo struct {
	Network        string `json:"network"`
	Broadcast      string `json:"broadcast"`
	LowLimitHost   string `json:"low_limit_host"`
	UpperLimitHost string `json:"upper_limit_host"`
	UsableHosts    int    `json:"usable_hosts"`
}

func (s SubnetInfo) RenderTable() (string, error) {
	return fmt.Sprintf(
		"Network IP Address: %s\nBroadcast IP Address: %s\nUsable Host Range: %s - %s \nUsable Hosts:  %d\n",
		s.Network, s.Broadcast, s.LowLimitHost, s.UpperLimitHost, s.UsableHosts,
	), nil
}

type SubnetInfoList []SubnetInfo

func (l SubnetInfoList) RenderTable() (string, error) {
	var b strings.Builder
	w := tabwriter.NewWriter(&b, 0, 0, 2, ' ', tabwriter.Debug)
	_, err := fmt.Fprintln(w, "SUBNET\tBROADCAST\tHOST RANGE\tUSABLE HOSTS")
	if err != nil {
		return "", err
	}
	for _, v := range l {
		if _, err := fmt.Fprintf(w, "%v\t%v\t%v - %v\t%v\n", v.Network, v.Broadcast, v.LowLimitHost, v.UpperLimitHost, v.UsableHosts); err != nil {
			return "", err
		}
	}
	err = w.Flush()
	if err != nil {
		return "", err
	}
	return b.String(), nil
}

func getNetworkAndMask(cidr string) (*net.IPNet, error) {
	// net.ParseCIDR valida eficientemente IPv4 e IPv6
	_, ipnet, err := net.ParseCIDR(cidr)

	if err != nil {
		return nil, fmt.Errorf("parsing CIDR %q: %w", cidr, err)
	}

	return ipnet, nil
}

func calculateHostAvailable(network net.IPNet) int {
	ones, _ := network.Mask.Size()
	hostBits := 32 - ones

	// /32 has only 1 address total, used as a single host route (e.g. loopback).
	if hostBits == 0 {
		return 1
	}
	// /31 has only 2 addresses total. Reserving one for network and one for
	// broadcast would leave zero usable hosts, so RFC 3021 makes both addresses
	// usable -- there is no separate network/broadcast address. Common for
	// point-to-point links between two routers.
	if hostBits == 1 {
		return 2
	}

	hosts := math.Pow(2, float64(hostBits)) - 2

	return int(hosts)
}

func calculateBroadcastAddress(network net.IPNet) (net.IP, error) {
	ip4 := network.IP.To4()
	mask4 := net.IP(network.Mask).To4()

	if ip4 == nil || mask4 == nil {
		return nil, fmt.Errorf("only IPv4 supported")
	}

	ipInt := binary.BigEndian.Uint32(ip4)
	maskInt := binary.BigEndian.Uint32(mask4)

	// Broadcast = IP OR (NOT Mask)
	broadcastInt := ipInt | (^maskInt)

	broadcastIP := make(net.IP, 4)
	binary.BigEndian.PutUint32(broadcastIP, broadcastInt)

	return broadcastIP, nil
}

func GetSubnetInfo(cidr string) (SubnetInfo, error) {
	network, err := getNetworkAndMask(cidr)
	if err != nil {
		return SubnetInfo{}, err
	}
	hosts := calculateHostAvailable(*network)
	broadcast, err := calculateBroadcastAddress(*network)
	if err != nil {
		return SubnetInfo{}, err
	}

	ones, _ := network.Mask.Size()
	hostBits := 32 - ones
	lowLimitHost, upperLimitHost := calculateUsableHostRange(network.IP, broadcast, hostBits)

	res := SubnetInfo{
		Network:        network.String(),
		Broadcast:      broadcast.String(),
		LowLimitHost:   lowLimitHost.String(),
		UpperLimitHost: upperLimitHost.String(),
		UsableHosts:    hosts,
	}

	return res, nil
}

func calculateAdditionalNetworkBits(desiredSubnets int) int {
	requiredBits := math.Log2(float64(desiredSubnets))
	return int(math.Ceil(requiredBits))
}

func calculateNewNetworkMask(currentMask net.IPMask, newBits int) net.IPMask {
	maskBites := binary.BigEndian.Uint32(currentMask)
	newNetworkMask := maskBites >> newBits // | (1<<(32-newBits))
	//fill empty bits with 1
	for i := 1; i <= newBits; i++ {
		newNetworkMask |= (1 << (32 - i))
	}

	maskBytes := make(net.IPMask, 4)
	binary.BigEndian.PutUint32(maskBytes, newNetworkMask)
	return maskBytes
}

func findInterestingOctet(mask net.IPMask) (index int, value byte, found bool) {
	for i, octeto := range mask {
		if octeto > 0 && octeto < 255 {
			return i, octeto, true
		}
	}
	return -1, 0, false
}

func calculateUsableHostRange(network net.IP, broadcast net.IP, hostBits int) (net.IP, net.IP) {
	// /31 y /32 no tienen direccion de red/broadcast separada: ambos extremos son usables
	if hostBits < 2 {
		return network, broadcast
	}

	networkInt := binary.BigEndian.Uint32(network.To4())
	broadcastInt := binary.BigEndian.Uint32(broadcast.To4())

	lowLimitHost := make(net.IP, 4)
	binary.BigEndian.PutUint32(lowLimitHost, networkInt+1)

	upperLimitHost := make(net.IP, 4)
	binary.BigEndian.PutUint32(upperLimitHost, broadcastInt-1)

	return lowLimitHost, upperLimitHost
}

func calculateSubnets(network net.IPNet, newNetworkMask net.IPMask, desiredSubnets int) ([]SubnetInfo, error) {
	index, value, found := findInterestingOctet(newNetworkMask)

	ones, _ := newNetworkMask.Size()
	hostBits := 32 - ones

	if !found {
		index = ones/8 - 1
		value = 255
	}

	jumpSize := 256 - int(value)

	_network := make(net.IP, len(network.IP))
	copy(_network, network.IP)

	_subnetsInfo := make([]SubnetInfo, desiredSubnets)

	for i := 1; i <= desiredSubnets; i++ {

		next_subnet_address := int(_network[index]) + jumpSize
		if next_subnet_address > 255 {
			next_subnet_address = 255
		}

		ipNet := net.IPNet{IP: _network, Mask: newNetworkMask}

		broadcast_address, err := calculateBroadcastAddress(ipNet)

		if err != nil {
			return nil, err
		}

		low_limit_host, upper_limit_host := calculateUsableHostRange(_network, broadcast_address, hostBits)

		host_available := calculateHostAvailable(ipNet)

		_subnetsInfo[i-1] = SubnetInfo{
			Network:        _network.String(),
			Broadcast:      broadcast_address.String(),
			LowLimitHost:   low_limit_host.String(),
			UpperLimitHost: upper_limit_host.String(),
			UsableHosts:    host_available,
		}

		_network[index] = byte(next_subnet_address)
	}

	return _subnetsInfo, nil
}

func SplitSubnet(cidr string, split int) ([]SubnetInfo, error) {
	network, err := getNetworkAndMask(cidr)
	if err != nil {
		return nil, err
	}

	newBits := calculateAdditionalNetworkBits(split)

	ones, _ := network.Mask.Size()
	if ones+newBits > 32 {
		return nil, fmt.Errorf("%s does not have enough host bits to split into %d subnets", cidr, split)
	}

	newNetworkMask := calculateNewNetworkMask(network.Mask, newBits)

	res, err := calculateSubnets(*network, newNetworkMask, split)
	if err != nil {
		return nil, err
	}

	return res, nil
}
