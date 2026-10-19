package cmd

import (
	"fmt"
	"os"

	"github.com/RobertoC117/sonarkube/cmd/probe"
	"github.com/spf13/cobra"
)

var output string

// rootCmd is the base command, run when the binary is invoked with no subcommand.
var rootCmd = &cobra.Command{
	Use:   "sonarkube",
	Short: "Network toolkit for subnet math and host probing",
	Long:  `sonarkube is a CLI toolkit for network engineers: calculate and split IPv4 subnets, and probe hosts over TCP, DNS, and ICMP.`,
	// No volcar el Usage completo ante un error de runtime; Cobra sigue
	// imprimiendo el mensaje de error (una sola vez) en stderr.
	SilenceUsage: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if output != "table" && output != "json" {
			return fmt.Errorf("invalid --output value %q: must be \"table\" or \"json\"", output)
		}
		return nil
	},
}

// Execute is the entry point called by main.go.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(probe.ProbeCmd)
	rootCmd.PersistentFlags().StringVarP(&output, "output", "o", "table", "Output format: table or json")
}
