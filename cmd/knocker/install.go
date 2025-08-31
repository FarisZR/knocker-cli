package main

import (
	"log"

	"github.com/kardianos/service"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the Knocker service",
	Long:  `This command installs the Knocker service on the host system.`,
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

		err = s.Install()
		if err != nil {
			log.Fatal(err)
		}

		log.Println("Service installed successfully.")
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}