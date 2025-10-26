package main

import (
	"runtime"

	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the Knocker service",
	Long:  `This command installs the Knocker service on the host system.`,
	Run: func(cmd *cobra.Command, args []string) {
		s, err := newServiceInstance(true)
		if err != nil {
			logger.Fatal(err)
		}

		if err := s.Install(); err != nil {
			logger.Fatal(err)
		}

		logger.Println("Service installed successfully.")

		switch runtime.GOOS {
		case "linux":
			logger.Println("Hint: use `systemctl --user enable --now knocker` to start the service immediately.")
		case "darwin":
			logger.Println("Hint: use `launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/knocker.plist` to load the agent.")
		}
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}
