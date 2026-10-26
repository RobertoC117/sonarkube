package probe

import (
	"fmt"

	"github.com/RobertoC117/sonarkube/cmd/probe/dns"
	"github.com/RobertoC117/sonarkube/cmd/probe/icmp"
	"github.com/RobertoC117/sonarkube/cmd/probe/tcp"
	"github.com/RobertoC117/sonarkube/cmd/probe/tls"
	"github.com/spf13/cobra"
)

// timeout is the shared per-probe timeout in milliseconds, exposed as a
// persistent flag so every subcommand inherits it.
var timeout int64

var ProbeCmd = &cobra.Command{
	Use:   "probe",
	Short: "Run network probes against a host",
	Long:  `Groups the network diagnostic subcommands: tcp (check port connectivity), dns (resolve a hostname), icmp (ping a host) and tls (inspect a certificate).`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if timeout <= 0 {
			return fmt.Errorf("--timeout must be greater than 0")
		}
		return nil
	},
}

// init wires the subcommands under probe and declares the shared --timeout flag.
// EnableTraverseRunHooks lets both root's and probe's PersistentPreRunE run, so
// validating the timeout here does not disable root's --output validation.
func init() {
	cobra.EnableTraverseRunHooks = true

	ProbeCmd.PersistentFlags().Int64VarP(&timeout, "timeout", "t", 5000, "Timeout in milliseconds per probe (per packet for icmp)")

	ProbeCmd.AddCommand(tcp.TcpCmd)
	ProbeCmd.AddCommand(dns.DnsCmd)
	ProbeCmd.AddCommand(icmp.IcmpCmd)
	ProbeCmd.AddCommand(tls.TlsCmd)
}
