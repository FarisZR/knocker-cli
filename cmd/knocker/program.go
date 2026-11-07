package main

import (
	"sync"
	"time"

	"github.com/FarisZR/knocker-cli/internal/api"
	internalService "github.com/FarisZR/knocker-cli/internal/service"
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

// run loads global settings and runs the core service until shutdown.
func (p *program) run(quit <-chan struct{}) {
	apiClient := api.NewClient(viper.GetString("api_url"), viper.GetString("api_key"))
	configuredCheckInterval := time.Duration(viper.GetInt("check_interval")) * time.Minute
	ipCheckURL := viper.GetString("ip_check_url")
	ttl := viper.GetInt("ttl")

	checkInterval := internalService.NormalizeCheckInterval(configuredCheckInterval)
	if checkInterval != configuredCheckInterval {
		logger.Printf("Invalid check interval detected, defaulting to %v.", checkInterval)
	}

	knockCadence := internalService.KnockCadenceFromTTL(ttl)
	watchNetwork := viper.GetBool("watch_network") || ipCheckURL != ""
	if ipCheckURL != "" {
		logger.Println("ip_check_url is deprecated; enabling server status checks and network monitoring. The URL will not be contacted.")
	}
	knockerService := internalService.NewService(apiClient, knockCadence, watchNetwork, checkInterval, ttl, "ttl", version, logger)

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
