package main

import (
	"github.com/kardianos/service"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Knocker service",
	Long:  `This command starts the installed Knocker service.`,
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

		err = s.Start()
		if err != nil {
			logger.Fatal(err)
		}

		logger.Println("Service started successfully.")
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
