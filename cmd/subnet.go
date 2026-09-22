package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"
	"github.com/RobertoC117/sonarkube/internal/subnetcalc"
	"github.com/spf13/cobra"
)

var (
	info    bool
	split	int16
)

func getSubnetInfo(cidr string) error {

	subnetInfo, err := subnetcalc.GetSubnetInfo(cidr)

	if err != nil {
		return fmt.Errorf("error getting network info: %w", err)
	}

	fmt.Println("Network IP Address:", subnetInfo.Network)
	fmt.Println("Broadcast IP Address:", subnetInfo.Broadcast)
	fmt.Printf("Usable Host Range: %v - %v \n", subnetInfo.LowLimitHost, subnetInfo.UpperLimitHost)
	fmt.Println("Usable Hosts: ", subnetInfo.UsableHosts)

	return nil
}


func splitSubnet(cidr string, split int) error {
	info, err:= subnetcalc.SplitSubnet(cidr, split)

	if err != nil {
		return fmt.Errorf("error splitting network: %w", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.Debug)
	_, err = fmt.Fprintln(w, "SUBNET\tBROADCAST\tHOST RANGE\tUSABLE HOSTS")

	if err != nil {
		return err
	}

	for _, v := range info {
		if _, err := fmt.Fprintf(w, "%v\t%v\t%v - %v\t%v\n", v.Network, v.Broadcast, v.LowLimitHost, v.UpperLimitHost, v.UsableHosts); err != nil {
			return err
		}
	}

	err = w.Flush()

	if err != nil {
		return err
	}

	return nil
}


var subnetCmd = &cobra.Command{
	Use:   "subnet",
	Short: "Provide info abour subnet",
	Long:  `Provide info abour subnet`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		CIDR := args[0]

		if info {
			if err :=  getSubnetInfo(CIDR); err != nil {
				return err
			}
		}

		if split > 0 {
			if err := splitSubnet(CIDR, int(split)); err != nil {
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
