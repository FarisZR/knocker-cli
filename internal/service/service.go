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
	Logger     *log.Logger
	stop       chan struct{}
	lastIP     string
	ipCheckURL string
	ttl        int
}

func NewService(apiClient *api.Client, ipGetter IPGetter, interval time.Duration, ipCheckURL string, ttl int, logger *log.Logger) *Service {
	return &Service{
		APIClient:  apiClient,
		IPGetter:   ipGetter,
		Interval:   interval,
		Logger:     logger,
		stop:       make(chan struct{}),
		ipCheckURL: ipCheckURL,
		ttl:        ttl,
	}
}

func (s *Service) Run(quit <-chan struct{}) {
	if s.ipCheckURL == "" {
		s.Logger.Printf("Service running. Knocking every %v (no ip_check_url set).", s.Interval)
	} else {
		s.Logger.Printf("Service running. Checking for IP changes every %v.", s.Interval)
	}
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
	// If no IP check URL is provided, just knock without checking.
	// The remote API will use the request's source IP.
	if s.ipCheckURL == "" {
		s.Logger.Println("Knocking without IP check...")
		if knockResponse, err := s.APIClient.Knock("", s.ttl); err != nil {
			s.Logger.Printf("Knock failed: %v", err)
		} else if knockResponse != nil {
			s.Logger.Printf("Successfully knocked. Whitelisted entry: %s (ttl: %d seconds)", knockResponse.WhitelistedEntry, knockResponse.ExpiresInSeconds)
		}
		return
	}

	// If an IP check URL is provided, perform the check and compare.
	ip, err := s.IPGetter.GetPublicIP(s.ipCheckURL)
	if err != nil {
		s.Logger.Printf("Error getting public IP: %v", err)
		return
	}

	if ip != s.lastIP {
		s.Logger.Printf("IP changed from %s to %s. Knocking...", s.lastIP, ip)
		if err := s.APIClient.HealthCheck(); err != nil {
			s.Logger.Printf("Health check failed: %v", err)
			return
		}
		if knockResponse, err := s.APIClient.Knock(ip, s.ttl); err != nil {
			s.Logger.Printf("Knock failed: %v", err)
			return
		} else if knockResponse != nil {
			s.Logger.Printf("Successfully knocked and updated IP. Whitelisted entry: %s (ttl: %d seconds)", knockResponse.WhitelistedEntry, knockResponse.ExpiresInSeconds)
		} else {
			s.Logger.Println("Successfully knocked and updated IP.")
		}
		s.lastIP = ip
	}
}
