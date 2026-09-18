package cmd

import (
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"os"
	"text/tabwriter"
	"github.com/spf13/cobra"
)

var (
	info    bool
	split	int16
	verbose bool
)

func getNetworkAndMask(cidr string) (*net.IPNet, error) {
	// net.ParseCIDR valida eficientemente IPv4 e IPv6
	_, ipnet, err := net.ParseCIDR(cidr)

	if err != nil {
		return nil, fmt.Errorf("parseando CIDR %q: %w", cidr, err)
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
	if(verbose) {
		fmt.Printf("IP (v4): %v | Mask (v4): %v\n", ip4, mask4)
	}

	if ip4 == nil || mask4 == nil {
		return nil, fmt.Errorf("solo se soporta IPv4")
	}

	ipInt := binary.BigEndian.Uint32(ip4)
	maskInt := binary.BigEndian.Uint32(mask4)
	if(verbose) {
		fmt.Printf("ipInt:    %032b (%d)\n", ipInt, ipInt)
		fmt.Printf("maskInt:  %032b (%d)\n", maskInt, maskInt)
		fmt.Printf("^maskInt: %032b\n", ^maskInt)
	}

	// Broadcast = IP OR (NOT Mask)
	broadcastInt := ipInt | (^maskInt)
	if(verbose) {
		fmt.Printf("broadcastInt: %032b (%d)\n", broadcastInt, broadcastInt)
	}

	broadcastIP := make(net.IP, 4)
	binary.BigEndian.PutUint32(broadcastIP, broadcastInt)
	if(verbose) {
		fmt.Printf("Broadcast IP: %v\n", broadcastIP)
	}

	return broadcastIP, nil
}

func getSubnetInfo(cidr string) error {
	network, err := getNetworkAndMask(cidr)
	if err != nil {
		return err
	}
	hosts := calculateHostAvailable(*network)
	broadcast, err := calculateBroadcastAddress(*network)
	if err != nil {
		return err
	}

	fmt.Println("Network IP Address:", network.String())
	fmt.Println("Broadcast IP Address:", broadcast.String())
	fmt.Println("Usable Hosts: ", hosts)
	return nil
}

func calculateAdditionalNetworkBits(desiredSubnets int) int {
	requiredBits := math.Log2(float64(desiredSubnets))
	return int(math.Ceil(requiredBits))
}

func calculateNewNetworkMask(currentMask net.IPMask, newBits int) net.IPMask {
	maskBites := binary.BigEndian.Uint32(currentMask)
	newNetworkMask := maskBites >> newBits// | (1<<(32-newBits))
	//fill empty bits with 1
	for i := 1 ; i <= newBits; i++ {
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

func calculateSubnets(network net.IPNet, newNetworkMask net.IPMask, desiredSubnets int) error {
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

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.Debug)
	_, err := fmt.Fprintln(w, "SUBNET\tBROADCAST\tHOST RANGE\tUSABLE HOSTS")
	
	if err != nil {
		return err
	}

	for i := 1; i <= desiredSubnets; i++ {
		
		next_subnet_address := int(_network[index]) + jumpSize
		if(next_subnet_address > 255){
			next_subnet_address = 255
		}
		
		// fmt.Printf("Subnet address: %v \n", _network)

		broadcast_address := make(net.IP, len(_network))
  		copy(broadcast_address, _network)
		
		broadcast_address[index] += byte(jumpSize -1)

		low_limit_host := make(net.IP, len(_network))
  		copy(low_limit_host, _network)

		upper_limit_host := make(net.IP, len(_network))
  		copy(upper_limit_host, broadcast_address)

		// /31 y /32 no tienen direccion de red/broadcast separada: ambos extremos son usables
		if hostBits >= 2 {
			low_limit_host[index] = _network[index] + 1
			upper_limit_host[index] = broadcast_address[index] - 1
		}

		ipNet := net.IPNet{IP: _network, Mask: newNetworkMask}
		host_available := calculateHostAvailable(ipNet)

		if _, err := fmt.Fprintf(w, "%v\t%v\t%v - %v\t%v\n", _network, broadcast_address, low_limit_host, upper_limit_host, host_available); err != nil {
			return err
		}
		_network[index] = byte(next_subnet_address)
	}

	err = w.Flush()

	if err != nil {
		return err
	}

	return nil
	// fmt.Printf("Value: %d | Jump size: %d | index: %v \n", value, jumpSize, index)
	// fmt.Println("Network: ", network)
}

func splitSubnet(cidr string, split int) error {
	network, err := getNetworkAndMask(cidr)
	if err != nil {
		return err
	}

	newBits := calculateAdditionalNetworkBits(split)

	ones, _ := network.Mask.Size()
	if ones+newBits > 32 {
		return fmt.Errorf("%s no tiene suficientes bits de host para dividirse en %d subredes", cidr, split)
	}

	newNetworkMask := calculateNewNetworkMask(network.Mask, newBits)

	err = calculateSubnets(*network, newNetworkMask, split)
	if err != nil {
		return err
	}

	return nil
}


var subnetCmd = &cobra.Command{
	Use:   "subnet",
	Short: "Provide info abour subnet",
	Long:  `Provide info abour subnet`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		CIDR := args[0]

		if info {
			if err := getSubnetInfo(CIDR); err != nil {
				return err
			}
		}

		if split > 0 {
			if err := splitSubnet(CIDR, int(split)); err != nil {
				return err
			}
		}

		return nil
	},
}

// init() se ejecuta al importar el paquete: aqui registramos el subcomando
// en el root y declaramos sus flags locales (solo existen para "sumar").
func init() {
	rootCmd.AddCommand(subnetCmd)
	subnetCmd.Flags().BoolVar(&info, "info", false, "Show info about subnet")
	subnetCmd.Flags().Int16Var(&split, "split", 0, "Split subnet")
	subnetCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
}
