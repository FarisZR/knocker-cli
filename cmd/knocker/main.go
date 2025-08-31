package main

import (
	"fmt"
	"os"

	"github.com/FarisZR/knocker-cli/internal/config"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "knocker",
	Short: "Knocker is a CLI tool to automatically manage IP whitelisting.",
	Long: `A reliable, cross-platform service that keeps your external IP address whitelisted.
It runs in the background, detects IP changes, and ensures you always have access.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Default command logic (e.g., show help)
		cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Whoops. There was an error while executing your CLI '%s'", err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(config.InitConfig)

	rootCmd.PersistentFlags().StringVar(&config.CfgFile, "config", "", "config file (default is $HOME/.knocker.yaml)")
}

func main() {
	Execute()
}