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
	p.quit = make(chan struct{})
	go p.run()
	return nil
}
func (p *program) run() {
	apiClient := api.NewClient(viper.GetString("api_url"), viper.GetString("api_key"))
	ipGetter := util.NewIPGetter()
	knockerService := internalService.NewService(apiClient, ipGetter, 5*time.Minute, "https://api.ipify.org")

	knockerService.Run(p.quit)
}
func (p *program) Stop(s service.Service) error {
	close(p.quit)
	return nil
}