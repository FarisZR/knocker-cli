package main

import (
	"errors"

	"github.com/kardianos/service"
	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall the Knocker service",
	Long:  `This command uninstalls the Knocker service from the host system.`,
	Run: func(cmd *cobra.Command, args []string) {
		s, err := newServiceInstance(false)
		if err != nil {
			logger.Fatal(err)
		}

		if stopErr := s.Stop(); stopErr != nil && !errors.Is(stopErr, service.ErrNotInstalled) {
			logger.Printf("Warning: could not stop service prior to uninstall: %v", stopErr)
		}

		if err := s.Uninstall(); err != nil {
			if errors.Is(err, service.ErrNotInstalled) {
				logger.Println("Service not installed, nothing to uninstall.")
				return
			}
			logger.Fatal(err)
		}

		logger.Println("Service uninstalled successfully.")
	},
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
}
