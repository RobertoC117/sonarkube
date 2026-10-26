package tls

import (
	"context"
	"time"

	"github.com/RobertoC117/sonarkube/internal/output"
	"github.com/RobertoC117/sonarkube/internal/probe"
	"github.com/spf13/cobra"
)

var insecure bool

var TlsCmd = &cobra.Command{
	Use:   "tls <host:port>",
	Short: "Inspect the TLS certificate presented by a host and port",
	Long:  `Performs a TLS handshake against <host:port> and reports the negotiated TLS version, the certificate issuer, the names it covers (SANs) and whether it is currently valid. Use --insecure to complete the handshake and inspect the certificate even when it fails validation (expired, self-signed or wrong hostname).`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]

		format, err := cmd.Flags().GetString("output")
		if err != nil {
			return err
		}

		timeout, err := cmd.Flags().GetInt64("timeout")
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Millisecond)
		defer cancel()

		cert, err := probe.TLSVerification(ctx, target, insecure, time.Now())
		if err != nil {
			return err
		}

		if err := output.Print(format, cert); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	TlsCmd.Flags().BoolVar(&insecure, "insecure", false, "Complete the handshake and inspect the certificate even if it fails validation (expired, self-signed, wrong hostname)")
}
