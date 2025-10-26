package main

import (
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the Knocker service",
	Long:  `This command stops the running Knocker service.`,
	Run: func(cmd *cobra.Command, args []string) {
		s, err := newServiceInstance(false)
		if err != nil {
			logger.Fatal(err)
		}

		if err := s.Stop(); err != nil {
			logger.Fatal(err)
		}

		logger.Println("Service stopped successfully.")
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
