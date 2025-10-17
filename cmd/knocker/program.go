package main

import (
	"time"

	"github.com/FarisZR/knocker-cli/internal/api"
	internalService "github.com/FarisZR/knocker-cli/internal/service"
	"github.com/FarisZR/knocker-cli/internal/util"
	"github.com/kardianos/service"
	"github.com/spf13/viper"
)

type program struct {
	quit chan struct{}
}

func (p *program) Start(s service.Service) error {
	logger.Println("Starting Knocker service...")
	p.quit = make(chan struct{})
	go p.run()
	return nil
}
func (p *program) run() {
	apiClient := api.NewClient(viper.GetString("api_url"), viper.GetString("api_key"))
	ipGetter := util.NewIPGetter()
	interval := time.Duration(viper.GetInt("interval")) * time.Minute
	ipCheckURL := viper.GetString("ip_check_url")
	ttl := viper.GetInt("ttl")

	// Perform initial health check
	if err := apiClient.HealthCheck(); err != nil {
		logger.Fatalf("Initial health check failed: %v. Please check your API URL and key.", err)
	}
	logger.Println("API health check successful.")

	knockerService := internalService.NewService(apiClient, ipGetter, interval, ipCheckURL, ttl, logger)

	knockerService.Run(p.quit)
}
func (p *program) Stop(s service.Service) error {
	logger.Println("Stopping Knocker service...")
	close(p.quit)
	return nil
}