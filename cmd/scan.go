package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	outfmt "github.com/RobertoC117/sonarkube/internal/output"
	"github.com/RobertoC117/sonarkube/internal/scan"
	targetsLib "github.com/RobertoC117/sonarkube/internal/targets"
	"github.com/spf13/cobra"
)

var (
	targets            string
	concurrency int
	scan_type          string
	timeout            int64
)

func probeFor(scanType string, timeout int64) (scan.PathFinderFunc, error) {
	switch scanType {
	case "tcp":
		return scan.TCPProbe(time.Duration(timeout) * time.Millisecond), nil // ya tiene la forma, va directo
	case "dns":
		return scan.DNSProbe(time.Duration(timeout) * time.Millisecond), nil // idem
	case "icmp":
		return scan.ICMPProbe(3, time.Duration(timeout)*time.Millisecond), nil // acá entra el adaptador
	default:
		return nil, fmt.Errorf("tipo inválido: %s", scanType)
	}
}

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {

		ctx := context.Background()

		file, err := os.Open(targets)

		if err != nil {
			return fmt.Errorf("error opening targets file: %w", err)
		}

		defer func() {
			_ = file.Close()
		}()

		parsedTargets, err := targetsLib.ParseTargets(file)

		if err != nil {
			return fmt.Errorf("error parsing targets: %w", err)
		}

		scanner, err := probeFor(scan_type, timeout)

		if err != nil {
			return err
		}

		connResults, err := scan.Scan(ctx, parsedTargets, concurrency, scanner)

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
	scanCmd.Flags().StringVar(&targets, "targets", "", "Usage description")
	scanCmd.Flags().IntVar(&concurrency, "concurrency", 50, "Usage description")
	scanCmd.Flags().StringVar(&scan_type, "type", "", "Usage description")
	scanCmd.Flags().Int64Var(&timeout, "timeout", scan.DefaultTimeout.Milliseconds(), "Timeout description in millis")
	rootCmd.AddCommand(scanCmd)
}
