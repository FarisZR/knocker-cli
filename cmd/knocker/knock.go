package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/FarisZR/knocker-cli/internal/api"
	"github.com/FarisZR/knocker-cli/internal/journald"
	internalService "github.com/FarisZR/knocker-cli/internal/service"
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
		ttl := viper.GetInt("ttl")

		if apiURL == "" || apiKey == "" {
			logger.Fatal("API URL and API Key must be configured.")
		}

		client := api.NewClient(apiURL, apiKey)

		logger.Println("Manually knocking to whitelist IP...")
		knockResponse, err := client.Knock("", ttl)
		if err != nil {
			emitManualKnockFailure(err)
			logger.Fatalf("Failed to knock: %v", err)
		}

		emitManualKnockSuccess(knockResponse)

		logger.Printf("Successfully knocked. Whitelisted entry: %s (ttl: %d seconds)", knockResponse.WhitelistedEntry, knockResponse.ExpiresInSeconds)
		fmt.Printf("Successfully knocked and whitelisted IP. TTL: %d seconds\n", knockResponse.ExpiresInSeconds)
	},
}

func init() {
	rootCmd.AddCommand(knockCmd)
}

func emitManualKnockFailure(err error) {
	msg := fmt.Sprintf("Manual knock failed: %v", err)
	_ = journald.Emit(internalService.EventKnockTriggered, msg, journald.PriErr, journald.Fields{
		"KNOCKER_TRIGGER_SOURCE": internalService.TriggerSourceCLI,
		"KNOCKER_RESULT":         internalService.ResultFailure,
	})
	_ = journald.Emit(internalService.EventError, msg, journald.PriErr, journald.Fields{
		"KNOCKER_ERROR_CODE": internalService.ErrorCodeKnockFailed,
		"KNOCKER_ERROR_MSG":  msg,
		"KNOCKER_CONTEXT":    "cli",
	})
}

func emitManualKnockSuccess(knockResponse *api.KnockResponse) {
	whitelistIP := ""
	ttlSeconds := 0
	expiresUnix := int64(0)
	if knockResponse != nil {
		whitelistIP = knockResponse.WhitelistedEntry
		ttlSeconds = knockResponse.ExpiresInSeconds
		expiresUnix = knockResponse.ExpiresAt
	}

	knockFields := journald.Fields{
		"KNOCKER_TRIGGER_SOURCE": internalService.TriggerSourceCLI,
		"KNOCKER_RESULT":         internalService.ResultSuccess,
	}
	if whitelistIP != "" {
		knockFields["KNOCKER_WHITELIST_IP"] = whitelistIP
	}
	_ = journald.Emit(internalService.EventKnockTriggered, "Manual knock succeeded", journald.PriInfo, knockFields)

	if knockResponse == nil {
		return
	}

	whitelistFields := journald.Fields{
		"KNOCKER_SOURCE": internalService.TriggerSourceCLI,
	}
	if whitelistIP != "" {
		whitelistFields["KNOCKER_WHITELIST_IP"] = whitelistIP
	}
	if ttlSeconds > 0 {
		whitelistFields["KNOCKER_TTL_SEC"] = strconv.Itoa(ttlSeconds)
	}
	if expiresUnix > 0 {
		whitelistFields["KNOCKER_EXPIRES_UNIX"] = strconv.FormatInt(expiresUnix, 10)
	}

	message := "Whitelist updated"
	if whitelistIP != "" && ttlSeconds > 0 && expiresUnix > 0 {
		message = fmt.Sprintf("Whitelisted %s for %ds (expires at %s)", whitelistIP, ttlSeconds, time.Unix(expiresUnix, 0).UTC().Format(time.RFC3339))
	} else if whitelistIP != "" {
		message = fmt.Sprintf("Whitelisted %s", whitelistIP)
	}

	_ = journald.Emit(internalService.EventWhitelistApplied, message, journald.PriInfo, whitelistFields)
}
