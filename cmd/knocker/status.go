package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Get the status of the Knocker service",
	Long:  `This command checks the status of the installed Knocker service.`,
	Run: func(cmd *cobra.Command, args []string) {
		s, err := newServiceInstance(false)
		if err != nil {
			logger.Fatal(err)
		}

		status, err := s.Status()
		if err != nil {
			logger.Fatal(err)
		}

		fmt.Printf("Service status: %v\n", status)
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
