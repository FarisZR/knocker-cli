package main

import (
	"time"

	"github.com/FarisZR/knocker-cli/internal/api"
	internalService "github.com/FarisZR/knocker-cli/internal/service"
	"github.com/FarisZR/knocker-cli/internal/util"
	"github.com/kardianos/service"
	"github.com/spf13/viper"
)

type program struct{}

func (p *program) Start(s service.Service) error {
	go p.run()
	return nil
}
func (p *program) run() {
	apiClient := api.NewClient(viper.GetString("api_url"), viper.GetString("api_key"))
	ipGetter := util.NewIPGetter()
	knockerService := internalService.NewService(apiClient, ipGetter, 5*time.Minute, "https://api.ipify.org")

	knockerService.Run()
}
func (p *program) Stop(s service.Service) error {
	return nil
}