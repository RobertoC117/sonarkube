package dns

import (
	"context"
	"fmt"
	"time"

	"github.com/RobertoC117/sonarkube/internal/output"
	"github.com/RobertoC117/sonarkube/internal/probe"
	"github.com/spf13/cobra"
)

var timeout int64

var DnsCmd = &cobra.Command{
	Use:   "dns <host>",
	Short: "Resolve a hostname via DNS",
	Long:  `Resolves <host> to its IP addresses using the system resolver and reports the resolution latency. Exits with a non-zero code if resolution does not complete within --timeout.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if timeout <= 0 {
			return fmt.Errorf("timeout must be greater than 0ms")
		}

		format, err := cmd.Flags().GetString("output")
		if err != nil {
			return err
		}

		dns := args[0]
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Millisecond)
		defer cancel()

		result, resolveErr := probe.ResolveDNS(ctx, dns)
		if err := output.Print(format, result); err != nil {
			return err
		}
		if resolveErr != nil {
			return resolveErr
		}
		return nil
	},
}

// init() se ejecuta al importar el paquete: aqui registramos el subcomando
// en el root y declaramos sus flags locales (solo existen para "sumar").
func init() {
	DnsCmd.Flags().Int64VarP(&timeout, "timeout", "t", 5000, "Timeout in milliseconds, by default is 5000")
}
