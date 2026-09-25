package probe

import (
	"github.com/RobertoC117/sonarkube/cmd/probe/dns"
	"github.com/RobertoC117/sonarkube/cmd/probe/icmp"
	"github.com/RobertoC117/sonarkube/cmd/probe/tcp"
	"github.com/spf13/cobra"
)

var ProbeCmd = &cobra.Command{
	Use:   "probe",
	Short: "Run network probes against a host",
	Long:  `Groups the network diagnostic subcommands: tcp (check port connectivity), dns (resolve a hostname), and icmp (ping a host).`,
}

// init() se ejecuta al importar el paquete: aqui registramos el subcomando
// en el root y declaramos sus flags locales (solo existen para "sumar").
func init() {
	ProbeCmd.AddCommand(tcp.TcpCmd)
	ProbeCmd.AddCommand(dns.DnsCmd)
	ProbeCmd.AddCommand(icmp.IcmpCmd)
}
