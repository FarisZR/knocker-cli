package main

import (
	"sync"
	"time"

	"github.com/FarisZR/knocker-cli/internal/api"
	internalService "github.com/FarisZR/knocker-cli/internal/service"
	"github.com/FarisZR/knocker-cli/internal/util"
	"github.com/kardianos/service"
	"github.com/spf13/viper"
)

type program struct {
	quit    chan struct{}
	mu      sync.RWMutex
	service *internalService.Service
}

func (p *program) Start(s service.Service) error {
	logger.Println("Starting Knocker service...")
	p.mu.Lock()
	p.quit = make(chan struct{})
	quit := p.quit
	p.mu.Unlock()

	go p.run(quit)
	return nil
}
func (p *program) run(quit <-chan struct{}) {
	apiClient := api.NewClient(viper.GetString("api_url"), viper.GetString("api_key"))
	ipGetter := util.NewIPGetter()
	configuredCheckInterval := time.Duration(viper.GetInt("check_interval")) * time.Minute
	ipCheckURL := viper.GetString("ip_check_url")
	ttl := viper.GetInt("ttl")

	checkInterval := internalService.NormalizeCheckInterval(configuredCheckInterval)
	if checkInterval != configuredCheckInterval {
		logger.Printf("Invalid check interval detected, defaulting to %v.", checkInterval)
	}

	knockCadence := internalService.KnockCadenceFromTTL(ttl)
	cadenceSource := "ttl"
	if ipCheckURL != "" {
		knockCadence = checkInterval
		cadenceSource = "check_interval"
	}

	// Perform initial health check
	if err := apiClient.HealthCheck(); err != nil {
		logger.Fatalf("Initial health check failed: %v. Please check your API URL and key.", err)
	}
	logger.Println("API health check successful.")

	knockerService := internalService.NewService(apiClient, ipGetter, knockCadence, ipCheckURL, ttl, cadenceSource, version, logger)

	p.mu.Lock()
	p.service = knockerService
	p.mu.Unlock()
	defer func() {
		p.mu.Lock()
		p.service = nil
		p.mu.Unlock()
	}()

	knockerService.Run(quit)
}
func (p *program) Stop(s service.Service) error {
	logger.Println("Stopping Knocker service...")
	p.mu.RLock()
	svc := p.service
	p.mu.RUnlock()
	if svc != nil {
		svc.NotifyStopping()
		svc.Stop()
	}

	p.mu.Lock()
	if p.quit != nil {
		close(p.quit)
		p.quit = nil
	}
	p.mu.Unlock()
	return nil
}
