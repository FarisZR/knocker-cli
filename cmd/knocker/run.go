package main

import (
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the Knocker service",
	Long:  `This command starts the Knocker service, which will run in the foreground.`,
	Run: func(cmd *cobra.Command, args []string) {
		s, err := newServiceInstance(false)
		if err != nil {
			logger.Fatal(err)
		}

		if err := s.Run(); err != nil {
			logger.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
