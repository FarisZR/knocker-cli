package main

import (
	"fmt"

	"github.com/FarisZR/knocker-cli/internal/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var knockCmd = &cobra.Command{
	Use:   "knock",
	Short: "Manually trigger a whitelist request",
	Long:  `Manually triggers a request to the Knocker API to whitelist the public IP of the machine.`,
	Run: func(cmd *cobra.Command, args []string) {
		apiURL := viper.GetString("api_url")
		apiKey := viper.GetString("api_key")

		if apiURL == "" || apiKey == "" {
			logger.Fatal("API URL and API Key must be configured.")
		}

		client := api.NewClient(apiURL, apiKey)

		logger.Println("Manually knocking to whitelist IP...")
		knockResponse, err := client.Knock("", 0)
		if err != nil {
			logger.Fatalf("Failed to knock: %v", err)
		}

		logger.Printf("Successfully knocked. Whitelisted entry: %s", knockResponse.WhitelistedEntry)
		fmt.Println("Successfully knocked and whitelisted IP.")
	},
}

func init() {
	rootCmd.AddCommand(knockCmd)
}