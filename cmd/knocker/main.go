package main

import (
	"fmt"
	"log"
	"os"

	"github.com/FarisZR/knocker-cli/internal/config"
	"github.com/kardianos/service"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	logger  *log.Logger
	version = "dev"
)

var rootCmd = &cobra.Command{
	Use:     "knocker",
	Version: version,
	Short:   "Knocker is a CLI tool to automatically manage IP whitelisting.",
	Long: `A reliable, cross-platform service that keeps your external IP address whitelisted.
It runs in the background, detects IP changes, and ensures you always have access.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		applyConfigDefaults(cmd, viper.GetViper())
		logger = log.New(os.Stdout, "knocker: ", log.LstdFlags)
	},
	Run: func(cmd *cobra.Command, args []string) {
		if !service.Interactive() {
			s, err := newServiceInstance(false)
			if err != nil {
				logger.Fatalf("unable to initialise service runtime: %v", err)
			}

			if err := s.Run(); err != nil {
				logger.Fatalf("service run failed: %v", err)
			}
			return
		}

		if err := cmd.Help(); err != nil {
			logger.Printf("unable to show help: %v", err)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Whoops. There was an error while executing your CLI '%s'", err)
		os.Exit(1)
	}
}

func applyConfigDefaults(cmd *cobra.Command, v *viper.Viper) {
	if cmd == nil {
		return
	}

	// Apply to current command flags.
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if !flag.Changed && v.IsSet(flag.Name) {
			cmd.Flags().Set(flag.Name, v.GetString(flag.Name))
		}
	})

	// Apply to persistent flags declared on this command.
	cmd.PersistentFlags().VisitAll(func(flag *pflag.Flag) {
		if !flag.Changed && v.IsSet(flag.Name) {
			cmd.PersistentFlags().Set(flag.Name, v.GetString(flag.Name))
		}
	})

	applyConfigDefaults(cmd.Parent(), v)
}

func init() {
	cobra.OnInitialize(config.InitConfig)
	rootCmd.SetVersionTemplate(`{{printf "%s\n" .Version}}`)

	rootCmd.PersistentFlags().StringVar(&config.CfgFile, "config", "", "config file (default is $HOME/.knocker.yaml)")
	rootCmd.PersistentFlags().Int("interval", 5, "Interval in minutes to check for IP changes")
	rootCmd.PersistentFlags().String("ip_check_url", "", "URL of the external IP checker service")
	rootCmd.PersistentFlags().Int("ttl", 0, "Time to live in seconds for the knock request (0 for server default)")
	viper.BindPFlag("interval", rootCmd.PersistentFlags().Lookup("interval"))
	viper.BindPFlag("ip_check_url", rootCmd.PersistentFlags().Lookup("ip_check_url"))
	viper.BindPFlag("ttl", rootCmd.PersistentFlags().Lookup("ttl"))
	viper.SetDefault("interval", 5)
	viper.SetDefault("ttl", 0)
}

func main() {
	Execute()
}
