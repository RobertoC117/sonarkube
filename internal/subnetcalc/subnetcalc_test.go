package subnetcalc

import (
	"net"
	"testing"
)

// Convenciones de Go testing que vas a ver en este archivo:
//
//  1. El archivo se llama <nombre>_test.go y vive junto al código que prueba.
//     `go test` lo detecta automáticamente por el sufijo "_test.go".
//  2. Cada función de test empieza con "Test" + Nombre, y recibe *testing.T.
//     Es la firma exacta que `go test` busca — si le falta el "Test" o el *testing.T,
//     no se ejecuta como test.
//  3. package subnetcalc (mismo paquete, no subnetcalc_test): así podemos probar
//     funciones no exportadas como calculateHostAvailable o findInterestingOctet.
//  4. Patrón "table-driven": una tabla (slice de structs) con casos de entrada/salida,
//     y un solo loop que los corre todos. Es EL estilo idiomático en Go para esto —
//     evita repetir el mismo cuerpo de test 10 veces.
//  5. t.Run(nombre, func) crea un "subtest": podés correr uno solo con
//     `go test -run TestCalculateHostAvailable/24_normal`, y si falla, el output
//     te dice exactamente cuál caso de la tabla fue.

func TestCalculateHostAvailable(t *testing.T) {
	// mustParseCIDR es un helper local (ver abajo) para no repetir el manejo
	// de error de ParseCIDR en cada caso de la tabla.
	tests := []struct {
		name string
		cidr string
		want int
	}{
		{name: "/24 caso normal", cidr: "192.168.1.0/24", want: 254},
		{name: "/30 red chica", cidr: "192.168.1.0/30", want: 2},
		{name: "/31 punto a punto (RFC 3021)", cidr: "192.168.1.0/31", want: 2},
		{name: "/32 host unico", cidr: "192.168.1.0/32", want: 1},
	}

	for _, tt := range tests {
		// Importante: capturamos tt como variable local del loop (Go 1.22+ ya
		// lo hace bien por defecto; en versiones viejas había que hacer
		// `tt := tt` acá para evitar que todos los subtests compartan la
		// última iteración). Revisa tu go.mod: si dice go 1.22 o más, no hace falta.
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
		// Este es el caso que expuso el bug 2 que corregiste: el "octeto
		// interesante" de una /10 es el segundo byte, no el ultimo. Si el
		// calculo de broadcast solo tocara un byte (como hacia el codigo viejo
		// en calculateSubnets), este test fallaria.
		{name: "/10 octeto interesante no es el ultimo", cidr: "10.0.0.0/10", want: "10.63.255.255"},
		{name: "/31 sin broadcast propio pero igual calculable", cidr: "10.0.0.0/31", want: "10.0.0.1"},
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
		network   string // se parsea con net.ParseIP, no CIDR
		broadcast string
		hostBits  int
		wantLow   string
		wantHigh  string
	}{
		{
			name:      "/24 caso normal",
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
			name:      "/32 unico host (hostBits == 0)",
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

// --- TODO: para que sigas tu solo, con el mismo patron ---
//
// TestGetNetworkAndMask
//   - caso CIDR valido: revisa que ipnet.String() sea el esperado.
//   - caso CIDR invalido (ej. "no-es-un-cidr"): revisa que err != nil.
//     (aqui la tabla necesita un campo `wantErr bool` ademas de `want`)
func TestGetNetworkAndMask(t *testing.T) {

	tests := []struct {
		name              string
		networkAddres     string // se parsea con net.ParseIP, no CIDR
		wantNetworkAdress string
		wantNetworkMask   string
		wantErr           bool
	}{
		{
			name:              "/24 caso normal",
			networkAddres:     "192.168.1.0/24",
			wantNetworkAdress: "192.168.1.0/24",
			wantNetworkMask:   "255.255.255.0",
		},
		{
			name:              "/8 con host bits no en cero (normaliza a 10.0.0.0)",
			networkAddres:     "10.0.0.1/8",
			wantNetworkAdress: "10.0.0.0/8",
			wantNetworkMask:   "255.0.0.0",
		},
		{
			name:              "/10 mascara con octeto intermedio",
			networkAddres:     "10.0.0.5/10",
			wantNetworkAdress: "10.0.0.0/10",
			wantNetworkMask:   "255.192.0.0",
		},
		{name: "CIDR invalido", networkAddres: "no-es-un-cidr", wantErr: true},
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
// TestCalculateNewNetworkMask prueba calculateNewNetworkMask(currentMask net.IPMask, newBits int) net.IPMask.
// net.CIDRMask(bits, 32) construye una mascara IPv4 (32 bits totales) a partir
// de la cantidad de bits de red -- es la forma idiomatica de armar un
// net.IPMask sin pasar por net.ParseCIDR.
func TestCalculateNewNetworkMask(t *testing.T) {
	tests := []struct {
		name        string
		currentMask net.IPMask
		newBits     int
		wantMask    string
	}{
		{name: "/8 + 2 bits nuevos = /10", currentMask: net.CIDRMask(8, 32), newBits: 2, wantMask: "255.192.0.0"},
		{name: "/24 + 2 bits nuevos = /26", currentMask: net.CIDRMask(24, 32), newBits: 2, wantMask: "255.255.255.192"},
		// newBits=0 simula split=1: no se piden subredes nuevas, la mascara no debe cambiar.
		{name: "/24 + 0 bits nuevos = /24 sin cambio", currentMask: net.CIDRMask(24, 32), newBits: 0, wantMask: "255.255.255.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateNewNetworkMask(tt.currentMask, tt.newBits)

			// mismo truco que en TestGetNetworkAndMask: net.IPMask es []byte,
			// convertirlo a net.IP da el .String() en formato punteado.
			if net.IP(got).String() != tt.wantMask {
				t.Errorf("calculateNewNetworkMask(%v, %d) = %s, want %s", tt.currentMask, tt.newBits, net.IP(got).String(), tt.wantMask)
			}
		})
	}
}

// TestFindInterestingOctet prueba findInterestingOctet(mask net.IPMask) (index int, value byte, found bool).
// El "octeto interesante" es el unico byte de la mascara que no es 0 ni 255 --
// el que tiene el limite de la subred adentro. Una mascara "redonda" como /24
// no tiene ninguno.
func TestFindInterestingOctet(t *testing.T) {
	tests := []struct {
		name      string
		mask      net.IPMask
		wantIndex int
		wantValue byte
		wantFound bool
	}{
		{name: "/24 mascara redonda, no hay octeto intermedio", mask: net.CIDRMask(24, 32), wantIndex: -1, wantValue: 0, wantFound: false},
		{name: "/10 octeto intermedio en index 1", mask: net.CIDRMask(10, 32), wantIndex: 1, wantValue: 192, wantFound: true},
		{name: "/26 octeto intermedio en index 3", mask: net.CIDRMask(26, 32), wantIndex: 3, wantValue: 192, wantFound: true},
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

// TestGetSubnetInfo prueba GetSubnetInfo(cidr string) (SubnetInfo, error) -- es
// un test de integracion: ya no llama a una funcion interna suelta, sino a la
// funcion exportada que combina varias (getNetworkAndMask, calculateBroadcastAddress,
// calculateUsableHostRange, calculateHostAvailable).
func TestGetSubnetInfo(t *testing.T) {
	tests := []struct {
		name    string
		cidr    string
		want    SubnetInfo
		wantErr bool
	}{
		{
			name: "/24 caso normal",
			cidr: "192.168.1.0/24",
			want: SubnetInfo{
				Network:        "192.168.1.0/24",
				Broadcast:      "192.168.1.255",
				LowLimitHost:   "192.168.1.1",
				UpperLimitHost: "192.168.1.254",
				UsableHosts:    254,
			},
		},
		{name: "CIDR invalido", cidr: "no-es-un-cidr", wantErr: true},
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

// TestSplitSubnet prueba SplitSubnet(cidr string, split int) ([]SubnetInfo, error).
// Es el test de regresion mas importante del archivo: 10.0.0.0/8 dividido en 4
// es exactamente el caso que expuso los dos bugs de aritmetica de 32 bits que
// corregiste (calculateUsableHostRange y el broadcast_address en calculateSubnets).
// Si alguno de los dos vuelve a romperse, este test lo detecta.
func TestSplitSubnet(t *testing.T) {
	t.Run("10.0.0.0/8 dividido en 4 subredes /10", func(t *testing.T) {
		got, err := SplitSubnet("10.0.0.0/8", 4)
		if err != nil {
			t.Fatalf("SplitSubnet devolvio error inesperado: %v", err)
		}
		if len(got) != 4 {
			t.Fatalf("SplitSubnet devolvio %d subredes, want 4", len(got))
		}

		// Network no lleva sufijo de mascara (ej. "/10"): a diferencia de
		// GetSubnetInfo, que arma el string desde un net.IPNet, calculateSubnets
		// arma _subnetsInfo[i].Network desde un net.IP suelto (_network.String()).
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
		// /31 solo tiene 1 bit de host: pedir 4 subredes necesita 2 bits nuevos,
		// y 31+2 > 32, asi que debe fallar en vez de producir mascaras invalidas.
		_, err := SplitSubnet("10.0.0.0/31", 4)
		if err == nil {
			t.Fatalf("SplitSubnet(10.0.0.0/31, 4) = nil error, want error")
		}
	})
}

// mustParseCIDR es un helper de test: falla el test inmediatamente (t.Fatalf)
// si el CIDR no parsea, en vez de repetir el `if err != nil` en cada caso de
// la tabla. t.Helper() le dice a Go que, si falla, reporte la linea del
// caller (tt.cidr) en vez de esta linea generica.
func mustParseCIDR(t *testing.T, cidr string) (net.IP, *net.IPNet) {
	t.Helper()

	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		t.Fatalf("net.ParseCIDR(%q) fallo: %v", cidr, err)
	}
	return ip, ipnet
}
