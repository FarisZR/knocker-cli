package main

import (
	"github.com/kardianos/service"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the Knocker service",
	Long:  `This command stops the running Knocker service.`,
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

		err = s.Stop()
		if err != nil {
			logger.Fatal(err)
		}

		logger.Println("Service stopped successfully.")
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
