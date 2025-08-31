package main

import (
	"log"

	"github.com/kardianos/service"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the Knocker service",
	Long:  `This command starts the Knocker service, which will run in the foreground.`,
	Run: func(cmd *cobra.Command, args []string) {
		svcConfig := &service.Config{
			Name:        "Knocker",
			DisplayName: "Knocker IP Whitelist Service",
			Description: "Automatically whitelists the external IP of this device.",
		}

		prg := &program{}
		s, err := service.New(prg, svcConfig)
		if err != nil {
			log.Fatal(err)
		}

		logger, err := s.Logger(nil)
		if err != nil {
			log.Fatal(err)
		}

		err = s.Run()
		if err != nil {
			logger.Error(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}