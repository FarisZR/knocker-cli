package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/FarisZR/knocker-cli/internal/api"
)

type Service struct {
	APIClient       *api.Client
	Cadence         time.Duration
	Logger          *log.Logger
	cadenceSrc      string
	stop            chan struct{}
	watchNetwork    bool
	checkInterval   time.Duration
	networkChanges  <-chan struct{}
	networkDebounce time.Duration
	retryDelay      time.Duration
	ttl             int

	version          string
	currentWhitelist *whitelistState
	nextKnockUnix    int64

	stopOnce     sync.Once
	shutdownOnce sync.Once
}

func NewService(apiClient *api.Client, cadence time.Duration, watchNetwork bool, checkInterval time.Duration, ttl int, cadenceSource string, version string, logger *log.Logger) *Service {
	return &Service{
		APIClient: apiClient, Cadence: NormalizeCheckInterval(cadence), Logger: logger,
		cadenceSrc: cadenceSource, stop: make(chan struct{}), watchNetwork: watchNetwork,
		checkInterval: NormalizeCheckInterval(checkInterval), ttl: ttl, version: version,
		networkDebounce: time.Second, retryDelay: 5 * time.Second,
	}
}

func (s *Service) Run(quit <-chan struct{}) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Logger.Printf("Service running. Renewal cadence: %v; network monitoring: %t.", s.Cadence, s.watchNetwork)
	s.emitServiceState(ServiceStateStarted)
	var changes <-chan struct{}
	if s.watchNetwork {
		changes = s.networkChanges
		if changes == nil {
			var err error
			changes, err = watchNetworkChanges(ctx, s.Logger)
			if err != nil {
				s.Logger.Printf("Network listener unavailable; using periodic status checks: %v", err)
			}
		}
	}
	// Renewals and status checks have independent timers. Status checks must not
	// postpone an existing renewal unless they actually result in a successful knock.
	renewal := time.NewTimer(s.Cadence)
	defer renewal.Stop()
	resetRenewal := func(delay time.Duration) {
		renewal.Reset(delay)
		s.updateNextKnock(time.Now().Add(delay))
		s.emitStatusSnapshot()
	}
	pending := time.NewTimer(time.Hour)
	pending.Stop()
	defer pending.Stop()
	var pendingC <-chan time.Time
	retryStatus := func() { pending.Reset(s.retryDelay); pendingC = pending.C }
	initialDelay := s.Cadence
	if s.watchNetwork {
		knocked, err := s.checkStatus()
		if err != nil {
			retryStatus()
		}
		if !knocked && s.currentWhitelist != nil {
			s.adjustCadenceForTTL(s.currentWhitelist.TTLSeconds)
		}
		initialDelay = s.Cadence
	} else if !s.checkAndKnock() {
		initialDelay = s.retryDelay
	}
	resetRenewal(initialDelay)
	var pollC <-chan time.Time
	if s.watchNetwork {
		poll := time.NewTicker(s.checkInterval)
		defer poll.Stop()
		pollC = poll.C
	}
	defer func() {
		s.clearNextKnock()
		s.emitStatusSnapshot()
		s.emitServiceState(ServiceStateStopped)
	}()
	for {
		select {
		case <-renewal.C:
			s.checkWhitelistExpiry(time.Now())
			if s.checkAndKnock() {
				resetRenewal(s.Cadence)
			} else {
				resetRenewal(s.retryDelay)
			}
		case _, ok := <-changes:
			if !ok {
				changes = nil
				continue
			}
			pending.Reset(s.networkDebounce)
			pendingC = pending.C
		case <-pollC:
			pending.Reset(0)
			pendingC = pending.C
		case <-pendingC:
			pendingC = nil
			knocked, err := s.checkStatus()
			if err != nil {
				retryStatus()
			}
			if knocked {
				resetRenewal(s.Cadence)
			}
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

func (s *Service) checkAndKnock() bool {
	response, err := s.performKnock("", TriggerSourceSchedule)
	if err != nil {
		s.Logger.Printf("Knock failed: %v", err)
		return false
	}
	s.Logger.Printf("Successfully knocked. Whitelisted entry: %s (ttl: %d seconds)", response.WhitelistedEntry, response.ExpiresInSeconds)
	return true
}

// checkStatus returns whether a successful knock requires rescheduling.
// Unknown status never authorizes a knock or changes the renewal schedule.
func (s *Service) checkStatus() (bool, error) {
	status, err := s.APIClient.Status()
	if err != nil {
		s.Logger.Printf("Whitelist status unknown: %v", err)
		s.emitError(ErrorCodeStatusCheck, fmt.Sprintf("Whitelist status unknown: %v", err), s.APIClient.BaseURL)
		return false, err
	}
	if status.Whitelisted {
		s.currentWhitelist = &whitelistState{IP: status.IPAddress, ExpiresUnix: *status.ExpiresAt, TTLSeconds: status.ExpiresInSeconds, Source: TriggerSourceExternal}
		s.emitStatusSnapshot()
		return false, nil
	}
	s.currentWhitelist = nil
	s.emitStatusSnapshot()
	_, err = s.performKnock("", TriggerSourceNetwork)
	return err == nil, err
}

func (s *Service) performKnock(ip, source string) (*api.KnockResponse, error) {
	knockResponse, err := s.APIClient.Knock(ip, s.ttl)
	if err != nil {
		s.emitKnockTriggered(source, ResultFailure, ip)
		s.emitError(ErrorCodeKnockFailed, fmt.Sprintf("Knock failed: %v", err), ip)
		return nil, err
	}

	whitelistIP := ip
	if knockResponse.WhitelistedEntry != "" {
		whitelistIP = knockResponse.WhitelistedEntry
	}
	s.emitKnockTriggered(source, ResultSuccess, whitelistIP)

	s.handleWhitelistResponse(knockResponse, source)

	return knockResponse, nil
}

func (s *Service) handleWhitelistResponse(knockResponse *api.KnockResponse, source string) {
	s.currentWhitelist = &whitelistState{
		IP:          knockResponse.WhitelistedEntry,
		ExpiresUnix: knockResponse.ExpiresAt,
		TTLSeconds:  knockResponse.ExpiresInSeconds,
		Source:      source,
	}

	s.adjustCadenceForTTL(knockResponse.ExpiresInSeconds)

	s.emitWhitelistApplied(knockResponse.WhitelistedEntry, knockResponse.ExpiresInSeconds, knockResponse.ExpiresAt, source)
	s.emitStatusSnapshot()
}

func (s *Service) adjustCadenceForTTL(ttlSeconds int) {
	if ttlSeconds <= 0 {
		return
	}

	newCadence := KnockCadenceFromTTL(ttlSeconds)
	if newCadence == s.Cadence {
		return
	}

	s.Cadence = newCadence
	s.cadenceSrc = "ttl_response"
	s.Logger.Printf("Adjusted knock cadence to %v based on server TTL (%ds).", newCadence, ttlSeconds)
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
