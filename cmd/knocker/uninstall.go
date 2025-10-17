package main

import (
	"github.com/kardianos/service"
	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall the Knocker service",
	Long:  `This command uninstalls the Knocker service from the host system.`,
	Run: func(cmd *cobra.Command, args []string) {
		svcConfig := &service.Config{
			Name:        "Knocker",
			DisplayName: "Knocker IP Whitelist Service",
			Description: "Automatically whitelists the external IP of this device.",
		}

		prg := &program{}
		s, err := service.New(prg, svcConfig)
		if err != nil {
			logger.Fatal(err)
		}

		err = s.Uninstall()
		if err != nil {
			logger.Fatal(err)
		}

		logger.Println("Service uninstalled successfully.")
	},
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
}
