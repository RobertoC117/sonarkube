package cmd

import (
	"fmt"

	outfmt "github.com/RobertoC117/sonarkube/internal/output"
	"github.com/RobertoC117/sonarkube/internal/subnetcalc"
	"github.com/spf13/cobra"
)

var (
	info  bool
	split int16
)

func getSubnetInfo(cidr, format string) error {

	subnetInfo, err := subnetcalc.GetSubnetInfo(cidr)

	if err != nil {
		return fmt.Errorf("error getting network info: %w", err)
	}

	return outfmt.Print(format, subnetInfo)
}

func splitSubnet(cidr string, split int, format string) error {
	info, err := subnetcalc.SplitSubnet(cidr, split)

	if err != nil {
		return fmt.Errorf("error splitting network: %w", err)
	}

	return outfmt.Print(format, subnetcalc.SubnetInfoList(info))
}

var subnetCmd = &cobra.Command{
	Use:   "subnet <CIDR>",
	Short: "Calculate or split an IPv4 subnet",
	Long:  `Given a CIDR block, shows its network/broadcast address and usable host range (--info), or splits it into smaller subnets (--split).`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		format, err := cmd.Flags().GetString("output")
		if err != nil {
			return err
		}

		CIDR := args[0]

		if info {
			if err := getSubnetInfo(CIDR, format); err != nil {
				return err
			}
		}

		if split > 0 {
			if err := splitSubnet(CIDR, int(split), format); err != nil {
				return err
			}
		}

		return nil
	},
}

// init() se ejecuta al importar el paquete: aqui registramos el subcomando
// en el root y declaramos sus flags locales (solo existen para "sumar").
func init() {
	rootCmd.AddCommand(subnetCmd)
	subnetCmd.Flags().BoolVar(&info, "info", false, "Show info about subnet")
	subnetCmd.Flags().Int16Var(&split, "split", 0, "Split subnet")
}
