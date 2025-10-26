package main

import (
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Knocker service",
	Long:  `This command starts the installed Knocker service.`,
	Run: func(cmd *cobra.Command, args []string) {
		s, err := newServiceInstance(false)
		if err != nil {
			logger.Fatal(err)
		}

		if err := s.Start(); err != nil {
			logger.Fatal(err)
		}

		logger.Println("Service started successfully.")
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
