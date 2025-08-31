package service

import (
	"log"
	"time"

	"github.com/FarisZR/knocker-cli/internal/api"
)

type IPGetter interface {
	GetPublicIP(url string) (string, error)
}

type Service struct {
	APIClient  *api.Client
	IPGetter   IPGetter
	Interval   time.Duration
	stop       chan struct{}
	lastIP     string
	ipCheckURL string
}

func NewService(apiClient *api.Client, ipGetter IPGetter, interval time.Duration, ipCheckURL string) *Service {
	return &Service{
		APIClient:  apiClient,
		IPGetter:   ipGetter,
		Interval:   interval,
		stop:       make(chan struct{}),
		ipCheckURL: ipCheckURL,
	}
}

func (s *Service) Run(quit <-chan struct{}) {
	ticker := time.NewTicker(s.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.checkAndKnock()
		case <-quit:
			return
		case <-s.stop:
			return
		}
	}
}

func (s *Service) Stop() {
	close(s.stop)
}

func (s *Service) checkAndKnock() {
	ip, err := s.IPGetter.GetPublicIP(s.ipCheckURL)
	if err != nil {
		log.Printf("Error getting public IP: %v", err)
		return
	}

	if ip != s.lastIP {
		log.Printf("IP changed from %s to %s", s.lastIP, ip)
		if err := s.APIClient.HealthCheck(); err != nil {
			log.Printf("Health check failed: %v", err)
			return
		}
		if _, err := s.APIClient.Knock(ip, 0); err != nil {
			log.Printf("Knock failed: %v", err)
			return
		}
		s.lastIP = ip
	}
}