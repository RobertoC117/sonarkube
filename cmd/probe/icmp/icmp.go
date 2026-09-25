package icmp

import (
	"context"
	"fmt"
	"time"

	"github.com/RobertoC117/sonarkube/internal/output"
	"github.com/RobertoC117/sonarkube/internal/probe"
	"github.com/spf13/cobra"
)

var timeout int64
var count int

var IcmpCmd = &cobra.Command{
	Use:   "icmp <host>",
	Short: "Ping a host using ICMP echo requests",
	Long:  `Sends --count ICMP echo requests to <host> and reports each reply plus a final summary (transmitted/received/packet loss). Requires administrator privileges to open a raw socket (run with sudo); --timeout applies per packet, not to the whole run.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if timeout <= 0 {
			return fmt.Errorf("timeout must be greater than 0ms")
		}

		format, err := cmd.Flags().GetString("output")
		if err != nil {
			return err
		}

		host := args[0]
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		stats, err := probe.MakePing(ctx, host, count, time.Duration(timeout)*time.Millisecond)
		if err != nil {
			return err
		}

		return output.Print(format, stats)
	},
}

// init() se ejecuta al importar el paquete: aqui registramos el subcomando
// en el root y declaramos sus flags locales (solo existen para "sumar").
func init() {
	IcmpCmd.Flags().Int64VarP(&timeout, "timeout", "t", 5000, "Timeout in milliseconds, by default is 5000")
	IcmpCmd.Flags().IntVarP(&count, "count", "c", 4, "Number of ICMP echo requests to send")
}
