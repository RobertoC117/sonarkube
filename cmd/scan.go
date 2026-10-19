package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	outfmt "github.com/RobertoC117/sonarkube/internal/output"
	"github.com/RobertoC117/sonarkube/internal/scan"
	targetsLib "github.com/RobertoC117/sonarkube/internal/targets"
	"github.com/spf13/cobra"
)

var (
	targets     string
	concurrency int
	scan_type   string
	timeout     int64
)

func probeFor(scanType string, timeout int64) (scan.PathFinderFunc, error) {
	switch scanType {
	case "tcp":
		return scan.TCPProbe(time.Duration(timeout) * time.Millisecond), nil // already matches PathFinderFunc
	case "dns":
		return scan.DNSProbe(time.Duration(timeout) * time.Millisecond), nil // same
	case "icmp":
		return scan.ICMPProbe(3, time.Duration(timeout)*time.Millisecond), nil // ICMP needs the adapter
	default:
		return nil, fmt.Errorf("invalid --type %q: must be \"tcp\", \"dns\", or \"icmp\"", scanType)
	}
}

var scanCmd = &cobra.Command{
	Use:   "scan --targets <file> --type <tcp|dns|icmp>",
	Short: "Probe many targets concurrently with a bounded worker pool",
	Long: `Reads a list of targets from a file and probes them all in parallel,
using at most --concurrency simultaneous probes, and prints one result per target.

The target file has one entry per line; blank lines are ignored. The entry format
depends on --type:
  tcp   host:port   (e.g. github.com:443)
  dns   hostname    (e.g. github.com)
  icmp  hostname/IP (e.g. 1.1.1.1; requires elevated privileges, run with sudo)

Each probe is aborted if it exceeds --timeout (per packet for icmp). Press Ctrl-C
to cancel a running scan cleanly: in-flight probes stop and the command exits.`,
	RunE: func(cmd *cobra.Command, args []string) error {

		// Cancel the context on SIGINT (Ctrl-C); it propagates to Scan and
		// stops the in-flight probes.
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()

		file, err := os.Open(targets)
		if err != nil {
			return fmt.Errorf("could not open targets file %q: %w", targets, err)
		}

		defer func() {
			_ = file.Close()
		}()

		parsedTargets, err := targetsLib.ParseTargets(file)
		if err != nil {
			return fmt.Errorf("could not read targets from %q: %w", targets, err)
		}
		if len(parsedTargets) == 0 {
			return fmt.Errorf("no targets found in %q", targets)
		}

		prober, err := probeFor(scan_type, timeout)
		if err != nil {
			return err
		}

		connResults, err := scan.Scan(ctx, parsedTargets, concurrency, prober)
		if err != nil {
			return err
		}

		if err := outfmt.Print(output, connResults); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	scanCmd.Flags().StringVar(&targets, "targets", "", "Path to a file with one target per line (required)")
	scanCmd.Flags().IntVar(&concurrency, "concurrency", 50, "Maximum number of probes running at the same time")
	scanCmd.Flags().StringVar(&scan_type, "type", "", "Probe type to run: tcp, dns, or icmp (required)")
	scanCmd.Flags().Int64Var(&timeout, "timeout", scan.DefaultTimeout.Milliseconds(), "Maximum wait per target in milliseconds (per packet for icmp); 0 disables it")

	// Required flags: Cobra reports a clear error before RunE if they are missing.
	_ = scanCmd.MarkFlagRequired("targets")
	_ = scanCmd.MarkFlagRequired("type")

	rootCmd.AddCommand(scanCmd)
}
