package service

import (
	"fmt"
	"log"
	"sync"
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

	version          string
	currentWhitelist *whitelistState
	nextKnockUnix    int64

	stopOnce     sync.Once
	shutdownOnce sync.Once
}

func NewService(apiClient *api.Client, ipGetter IPGetter, interval time.Duration, ipCheckURL string, ttl int, version string, logger *log.Logger) *Service {
	return &Service{
		APIClient:  apiClient,
		IPGetter:   ipGetter,
		Interval:   interval,
		Logger:     logger,
		stop:       make(chan struct{}),
		ipCheckURL: ipCheckURL,
		ttl:        ttl,
		version:    version,
	}
}

func (s *Service) Run(quit <-chan struct{}) {
	if s.ipCheckURL == "" {
		s.Logger.Printf("Service running. Knocking every %v (no ip_check_url set).", s.Interval)
	} else {
		s.Logger.Printf("Service running. Checking for IP changes every %v.", s.Interval)
	}

	s.emitServiceState(ServiceStateStarted)
	s.updateNextKnock(time.Now().Add(s.Interval))
	s.emitStatusSnapshot()

	ticker := time.NewTicker(s.Interval)
	defer ticker.Stop()
	defer func() {
		s.clearNextKnock()
		s.emitStatusSnapshot()
		s.emitServiceState(ServiceStateStopped)
	}()

	for {
		select {
		case <-ticker.C:
			s.checkWhitelistExpiry(time.Now())
			s.checkAndKnock()
			s.updateNextKnock(time.Now().Add(s.Interval))
		case <-quit:
			s.NotifyStopping()
			s.checkWhitelistExpiry(time.Now())
			return
		case <-s.stop:
			s.NotifyStopping()
			s.checkWhitelistExpiry(time.Now())
			return
		}
	}
}

func (s *Service) Stop() {
	s.NotifyStopping()
	s.stopOnce.Do(func() {
		close(s.stop)
	})
}

func (s *Service) NotifyStopping() {
	s.shutdownOnce.Do(func() {
		s.emitServiceState(ServiceStateStopping)
	})
}

func (s *Service) checkAndKnock() {
	if s.ipCheckURL == "" {
		s.Logger.Println("Knocking without IP check...")
		knockResponse, err := s.performKnock("", TriggerSourceSchedule)
		if err != nil {
			s.Logger.Printf("Knock failed: %v", err)
			return
		}
		if knockResponse != nil {
			s.Logger.Printf("Successfully knocked. Whitelisted entry: %s (ttl: %d seconds)", knockResponse.WhitelistedEntry, knockResponse.ExpiresInSeconds)
		}
		return
	}

	ip, err := s.IPGetter.GetPublicIP(s.ipCheckURL)
	if err != nil {
		s.Logger.Printf("Error getting public IP: %v", err)
		s.emitError(ErrorCodeIPLookup, fmt.Sprintf("Error getting public IP: %v", err), s.ipCheckURL)
		return
	}

	if ip == s.lastIP {
		return
	}

	s.Logger.Printf("IP changed from %s to %s. Knocking...", s.lastIP, ip)

	if err := s.APIClient.HealthCheck(); err != nil {
		s.Logger.Printf("Health check failed: %v", err)
		s.emitError(ErrorCodeHealthCheck, fmt.Sprintf("Health check failed: %v", err), s.APIClient.BaseURL)
		return
	}

	knockResponse, err := s.performKnock(ip, TriggerSourceSchedule)
	if err != nil {
		s.Logger.Printf("Knock failed: %v", err)
		return
	}

	if knockResponse != nil {
		s.Logger.Printf("Successfully knocked and updated IP. Whitelisted entry: %s (ttl: %d seconds)", knockResponse.WhitelistedEntry, knockResponse.ExpiresInSeconds)
	} else {
		s.Logger.Println("Successfully knocked and updated IP.")
	}

	s.lastIP = ip
}

func (s *Service) performKnock(ip, source string) (*api.KnockResponse, error) {
	knockResponse, err := s.APIClient.Knock(ip, s.ttl)
	if err != nil {
		s.emitKnockTriggered(source, ResultFailure, ip)
		s.emitError(ErrorCodeKnockFailed, fmt.Sprintf("Knock failed: %v", err), ip)
		return nil, err
	}

	whitelistIP := ip
	if knockResponse != nil && knockResponse.WhitelistedEntry != "" {
		whitelistIP = knockResponse.WhitelistedEntry
	}
	s.emitKnockTriggered(source, ResultSuccess, whitelistIP)

	s.handleWhitelistResponse(knockResponse, source)

	return knockResponse, nil
}

func (s *Service) handleWhitelistResponse(knockResponse *api.KnockResponse, source string) {
	if knockResponse == nil {
		return
	}

	s.currentWhitelist = &whitelistState{
		IP:          knockResponse.WhitelistedEntry,
		ExpiresUnix: knockResponse.ExpiresAt,
		TTLSeconds:  knockResponse.ExpiresInSeconds,
		Source:      source,
	}

	s.emitWhitelistApplied(knockResponse.WhitelistedEntry, knockResponse.ExpiresInSeconds, knockResponse.ExpiresAt, source)
	s.emitStatusSnapshot()
}

func (s *Service) checkWhitelistExpiry(now time.Time) {
	if s.currentWhitelist == nil {
		return
	}

	if s.currentWhitelist.ExpiresUnix <= 0 {
		return
	}

	if now.Unix() < s.currentWhitelist.ExpiresUnix {
		return
	}

	ip := s.currentWhitelist.IP
	expiredUnix := s.currentWhitelist.ExpiresUnix

	s.currentWhitelist = nil

	if ip != "" && expiredUnix > 0 {
		s.Logger.Printf("Whitelist expired for %s at %s", ip, time.Unix(expiredUnix, 0).UTC().Format(time.RFC3339))
	} else if ip != "" {
		s.Logger.Printf("Whitelist expired for %s", ip)
	} else {
		s.Logger.Println("Whitelist expired")
	}

	s.emitWhitelistExpired(ip, expiredUnix)
	s.emitStatusSnapshot()
}
