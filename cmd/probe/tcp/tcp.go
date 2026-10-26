package tcp

import (
	"context"
	"time"

	"github.com/RobertoC117/sonarkube/internal/output"
	"github.com/RobertoC117/sonarkube/internal/probe"
	"github.com/spf13/cobra"
)

var TcpCmd = &cobra.Command{
	Use:   "tcp <host:port>",
	Short: "Check TCP connectivity to a host and port",
	Long:  `Attempts a TCP handshake against <host:port> and reports success and latency. Exits with a non-zero code if the connection could not be established within --timeout.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		format, err := cmd.Flags().GetString("output")
		if err != nil {
			return err
		}

		timeout, err := cmd.Flags().GetInt64("timeout")
		if err != nil {
			return err
		}

		hostAndPort := args[0]
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Millisecond)
		defer cancel()
		result, resolveErr := probe.EstablishTCPConnection(ctx, hostAndPort)

		if err := output.Print(format, result); err != nil {
			return err
		}

		if resolveErr != nil {
			return resolveErr
		}

		return nil
	},
}
