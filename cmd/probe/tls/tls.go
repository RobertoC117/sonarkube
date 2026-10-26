package tls

import (
	"context"
	"fmt"
	"time"

	"github.com/RobertoC117/sonarkube/internal/output"
	"github.com/RobertoC117/sonarkube/internal/probe"
	"github.com/spf13/cobra"
)

var (
	timeout  int
	insecure bool
)

var TlsCmd = &cobra.Command{
	Use:   "tls <host:port>",
	Short: "Inspect the TLS certificate presented by a host and port",
	Long:  `Performs a TLS handshake against <host:port> and reports the negotiated TLS version, the certificate issuer, the names it covers (SANs) and whether it is currently valid. Use --insecure to complete the handshake and inspect the certificate even when it fails validation (expired, self-signed or wrong hostname).`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {

		if timeout <= 0 {
			return fmt.Errorf("--timeout must be greater than 0")
		}

		target := args[0]

		format, flagErr := cmd.Flags().GetString("output")

		if flagErr != nil {
			return flagErr
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
	TlsCmd.Flags().IntVar(&timeout, "timeout", 5000, "Handshake timeout in milliseconds")
	TlsCmd.Flags().BoolVar(&insecure, "insecure", false, "Complete the handshake and inspect the certificate even if it fails validation (expired, self-signed, wrong hostname)")
}
