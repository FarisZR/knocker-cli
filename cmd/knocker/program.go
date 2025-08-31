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
	interval := time.Duration(viper.GetInt("interval")) * time.Minute
	ipCheckURL := viper.GetString("ip_check_url")
	knockerService := internalService.NewService(apiClient, ipGetter, interval, ipCheckURL)

	knockerService.Run(p.quit)
}
func (p *program) Stop(s service.Service) error {
	close(p.quit)
	return nil
}