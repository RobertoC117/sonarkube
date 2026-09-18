package cmd

import (
	"fmt"
	"os"
	"github.com/spf13/cobra"
)

// rootCmd es el comando base: se ejecuta cuando llamas al binario sin subcomandos.
var rootCmd = &cobra.Command{
	Use:   "sonarkube",
	Short: "No description provided",
	Long: `No description provided`,
}

// Execute es el punto de entrada que llama main.go.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
