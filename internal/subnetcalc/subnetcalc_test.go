package subnetcalc

import (
	"net"
	"testing"
)

// Go testing conventions used in this file:
//
//  1. The file is named <name>_test.go and lives next to the code it tests.
//     `go test` finds it automatically by the "_test.go" suffix.
//  2. Each test function starts with "Test" + Name and takes *testing.T. That
//     exact signature is what `go test` looks for.
//  3. package subnetcalc (same package, not subnetcalc_test): lets us test
//     unexported functions like calculateHostAvailable or findInterestingOctet.
//  4. Table-driven pattern: a table (slice of structs) of input/output cases and
//     a single loop running them all. The idiomatic Go style for this.
//  5. t.Run(name, func) creates a subtest: run one with
//     `go test -run TestCalculateHostAvailable/24_normal`; on failure the output
//     names exactly which table case failed.

func TestCalculateHostAvailable(t *testing.T) {
	// mustParseCIDR is a local helper (see below) to avoid repeating ParseCIDR
	// error handling in each table case.
	tests := []struct {
		name string
		cidr string
		want int
	}{
		{name: "/24 normal case", cidr: "192.168.1.0/24", want: 254},
		{name: "/30 small network", cidr: "192.168.1.0/30", want: 2},
		{name: "/31 punto a punto (RFC 3021)", cidr: "192.168.1.0/31", want: 2},
		{name: "/32 single host", cidr: "192.168.1.0/32", want: 1},
	}

	for _, tt := range tests {
		// Capture tt as a loop-local variable (Go 1.22+ does this by default;
		// older versions needed `tt := tt` here so subtests did not share the
		// last iteration). Check go.mod: for go 1.22 or later it is unnecessary.
		t.Run(tt.name, func(t *testing.T) {
			_, ipnet := mustParseCIDR(t, tt.cidr)

			got := calculateHostAvailable(*ipnet)

			if got != tt.want {
				t.Errorf("calculateHostAvailable(%s) = %d, want %d", tt.cidr, got, tt.want)
			}
		})
	}
}

func TestCalculateBroadcastAddress(t *testing.T) {
	tests := []struct {
		name string
		cidr string
		want string
	}{
		{name: "/24 normal", cidr: "192.168.1.0/24", want: "192.168.1.255"},
		// This is the case that exposed bug 2: the "interesting octet" of a /10
		// is the second byte, not the last. If broadcast calculation touched
		// only one byte (as the old calculateSubnets code did), this test would
		// fail.
		{name: "/10 interesting octet is not the last", cidr: "10.0.0.0/10", want: "10.63.255.255"},
		{name: "/31 without its own broadcast but still computable", cidr: "10.0.0.0/31", want: "10.0.0.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ipnet := mustParseCIDR(t, tt.cidr)

			got, err := calculateBroadcastAddress(*ipnet)
			if err != nil {
				t.Fatalf("calculateBroadcastAddress(%s) devolvio error inesperado: %v", tt.cidr, err)
			}

			if got.String() != tt.want {
				t.Errorf("calculateBroadcastAddress(%s) = %s, want %s", tt.cidr, got.String(), tt.want)
			}
		})
	}
}

func TestCalculateUsableHostRange(t *testing.T) {
	tests := []struct {
		name      string
		network   string // parsed with net.ParseIP, not CIDR
		broadcast string
		hostBits  int
		wantLow   string
		wantHigh  string
	}{
		{
			name:      "/24 normal case",
			network:   "192.168.1.0",
			broadcast: "192.168.1.255",
			hostBits:  8,
			wantLow:   "192.168.1.1",
			wantHigh:  "192.168.1.254",
		},
		{
			name:      "/31 ambos extremos usables (hostBits < 2)",
			network:   "10.0.0.0",
			broadcast: "10.0.0.1",
			hostBits:  1,
			wantLow:   "10.0.0.0",
			wantHigh:  "10.0.0.1",
		},
		{
			name:      "/32 single host (hostBits == 0)",
			network:   "10.0.0.5",
			broadcast: "10.0.0.5",
			hostBits:  0,
			wantLow:   "10.0.0.5",
			wantHigh:  "10.0.0.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			network := net.ParseIP(tt.network)
			broadcast := net.ParseIP(tt.broadcast)

			low, high := calculateUsableHostRange(network, broadcast, tt.hostBits)

			if low.String() != tt.wantLow {
				t.Errorf("low limit = %s, want %s", low.String(), tt.wantLow)
			}
			if high.String() != tt.wantHigh {
				t.Errorf("high limit = %s, want %s", high.String(), tt.wantHigh)
			}
		})
	}
}

// --- TODO: continue on your own with the same pattern ---
//
// TestGetNetworkAndMask
//   - valid CIDR case: check ipnet.String() is as expected.
//   - invalid CIDR case (e.g. "no-es-un-cidr"): check err != nil.
//     (the table then needs a `wantErr bool` field besides `want`)
func TestGetNetworkAndMask(t *testing.T) {

	tests := []struct {
		name              string
		networkAddres     string // parsed with net.ParseIP, not CIDR
		wantNetworkAdress string
		wantNetworkMask   string
		wantErr           bool
	}{
		{
			name:              "/24 normal case",
			networkAddres:     "192.168.1.0/24",
			wantNetworkAdress: "192.168.1.0/24",
			wantNetworkMask:   "255.255.255.0",
		},
		{
			name:              "/8 with host bits set (normalizes to 10.0.0.0)",
			networkAddres:     "10.0.0.1/8",
			wantNetworkAdress: "10.0.0.0/8",
			wantNetworkMask:   "255.0.0.0",
		},
		{
			name:              "/10 mask with middle octet",
			networkAddres:     "10.0.0.5/10",
			wantNetworkAdress: "10.0.0.0/10",
			wantNetworkMask:   "255.192.0.0",
		},
		{name: "invalid CIDR", networkAddres: "no-es-un-cidr", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			network, err := getNetworkAndMask(tt.networkAddres)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("getNetworkAndMask(%s) = nil error, want error", tt.networkAddres)
				}
				return
			}
			if err != nil {
				t.Fatalf("getNetworkAndMask(%s) devolvio error inesperado: %v", tt.networkAddres, err)
			}

			if network.String() != tt.wantNetworkAdress {
				t.Errorf("network address = %s, want %s", network.String(), tt.wantNetworkAdress)
			}
			if net.IP(network.Mask).String() != tt.wantNetworkMask {
				t.Errorf("network mask = %s, want %s", net.IP(network.Mask).String(), tt.wantNetworkMask)
			}
		})
	}
}

// TestCalculateNewNetworkMask tests calculateNewNetworkMask(currentMask net.IPMask, newBits int) net.IPMask.
// net.CIDRMask(bits, 32) builds an IPv4 mask (32 bits total) from the network
// bit count -- the idiomatic way to make a net.IPMask without net.ParseCIDR.
func TestCalculateNewNetworkMask(t *testing.T) {
	tests := []struct {
		name        string
		currentMask net.IPMask
		newBits     int
		wantMask    string
	}{
		{name: "/8 + 2 bits nuevos = /10", currentMask: net.CIDRMask(8, 32), newBits: 2, wantMask: "255.192.0.0"},
		{name: "/24 + 2 bits nuevos = /26", currentMask: net.CIDRMask(24, 32), newBits: 2, wantMask: "255.255.255.192"},
		// newBits=0 simulates split=1: no new subnets requested, mask must not change.
		{name: "/24 + 0 new bits = /24 unchanged", currentMask: net.CIDRMask(24, 32), newBits: 0, wantMask: "255.255.255.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateNewNetworkMask(tt.currentMask, tt.newBits)

			// same trick as in TestGetNetworkAndMask: net.IPMask is []byte,
			// converting it to net.IP gives .String() in dotted format.
			if net.IP(got).String() != tt.wantMask {
				t.Errorf("calculateNewNetworkMask(%v, %d) = %s, want %s", tt.currentMask, tt.newBits, net.IP(got).String(), tt.wantMask)
			}
		})
	}
}

// TestFindInterestingOctet tests findInterestingOctet(mask net.IPMask) (index int, value byte, found bool).
// The "interesting octet" is the only mask byte that is neither 0 nor 255 --
// the one holding the subnet boundary. A "round" mask like /24 has none.
func TestFindInterestingOctet(t *testing.T) {
	tests := []struct {
		name      string
		mask      net.IPMask
		wantIndex int
		wantValue byte
		wantFound bool
	}{
		{name: "/24 round mask, no middle octet", mask: net.CIDRMask(24, 32), wantIndex: -1, wantValue: 0, wantFound: false},
		{name: "/10 middle octet at index 1", mask: net.CIDRMask(10, 32), wantIndex: 1, wantValue: 192, wantFound: true},
		{name: "/26 middle octet at index 3", mask: net.CIDRMask(26, 32), wantIndex: 3, wantValue: 192, wantFound: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			index, value, found := findInterestingOctet(tt.mask)

			if index != tt.wantIndex || value != tt.wantValue || found != tt.wantFound {
				t.Errorf("findInterestingOctet(%v) = (%d, %d, %v), want (%d, %d, %v)",
					tt.mask, index, value, found, tt.wantIndex, tt.wantValue, tt.wantFound)
			}
		})
	}
}

// TestGetSubnetInfo tests GetSubnetInfo(cidr string) (SubnetInfo, error) -- an
// integration test: it calls the exported function that combines several
// (getNetworkAndMask, calculateBroadcastAddress, calculateUsableHostRange,
// calculateHostAvailable) rather than a single internal one.
func TestGetSubnetInfo(t *testing.T) {
	tests := []struct {
		name    string
		cidr    string
		want    SubnetInfo
		wantErr bool
	}{
		{
			name: "/24 normal case",
			cidr: "192.168.1.0/24",
			want: SubnetInfo{
				Network:        "192.168.1.0/24",
				Broadcast:      "192.168.1.255",
				LowLimitHost:   "192.168.1.1",
				UpperLimitHost: "192.168.1.254",
				UsableHosts:    254,
			},
		},
		{name: "invalid CIDR", cidr: "no-es-un-cidr", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetSubnetInfo(tt.cidr)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("GetSubnetInfo(%s) = nil error, want error", tt.cidr)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetSubnetInfo(%s) devolvio error inesperado: %v", tt.cidr, err)
			}
			if got != tt.want {
				t.Errorf("GetSubnetInfo(%s) = %+v, want %+v", tt.cidr, got, tt.want)
			}
		})
	}
}

// TestSplitSubnet tests SplitSubnet(cidr string, split int) ([]SubnetInfo, error).
// The most important regression test here: 10.0.0.0/8 split into 4 is exactly
// the case that exposed the two 32-bit arithmetic bugs (calculateUsableHostRange
// and the broadcast address in calculateSubnets). It catches either regressing.
func TestSplitSubnet(t *testing.T) {
	t.Run("10.0.0.0/8 dividido en 4 subredes /10", func(t *testing.T) {
		got, err := SplitSubnet("10.0.0.0/8", 4)
		if err != nil {
			t.Fatalf("SplitSubnet devolvio error inesperado: %v", err)
		}
		if len(got) != 4 {
			t.Fatalf("SplitSubnet devolvio %d subredes, want 4", len(got))
		}

		// Network carries no mask suffix (e.g. "/10"): unlike GetSubnetInfo,
		// which builds the string from a net.IPNet, calculateSubnets builds
		// _subnetsInfo[i].Network from a bare net.IP (_network.String()).
		want := []SubnetInfo{
			{Network: "10.0.0.0", Broadcast: "10.63.255.255", LowLimitHost: "10.0.0.1", UpperLimitHost: "10.63.255.254", UsableHosts: 4194302},
			{Network: "10.64.0.0", Broadcast: "10.127.255.255", LowLimitHost: "10.64.0.1", UpperLimitHost: "10.127.255.254", UsableHosts: 4194302},
			{Network: "10.128.0.0", Broadcast: "10.191.255.255", LowLimitHost: "10.128.0.1", UpperLimitHost: "10.191.255.254", UsableHosts: 4194302},
			{Network: "10.192.0.0", Broadcast: "10.255.255.255", LowLimitHost: "10.192.0.1", UpperLimitHost: "10.255.255.254", UsableHosts: 4194302},
		}

		for i, w := range want {
			if got[i] != w {
				t.Errorf("subred %d = %+v, want %+v", i, got[i], w)
			}
		}
	})

	t.Run("no hay bits suficientes para el split", func(t *testing.T) {
		// /31 has only 1 host bit: 4 subnets need 2 new bits, and 31+2 > 32, so
		// it must fail instead of producing invalid masks.
		_, err := SplitSubnet("10.0.0.0/31", 4)
		if err == nil {
			t.Fatalf("SplitSubnet(10.0.0.0/31, 4) = nil error, want error")
		}
	})
}

// mustParseCIDR is a test helper: it fails the test immediately (t.Fatalf) if
// the CIDR does not parse, instead of repeating `if err != nil` in each table
// case. t.Helper() tells Go to report the caller's line (tt.cidr) on failure
// rather than this generic line.
func mustParseCIDR(t *testing.T, cidr string) (net.IP, *net.IPNet) {
	t.Helper()

	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		t.Fatalf("net.ParseCIDR(%q) fallo: %v", cidr, err)
	}
	return ip, ipnet
}
