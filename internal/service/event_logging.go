package service

import (
	"fmt"
	"strconv"
	"time"

	"github.com/FarisZR/knocker-cli/internal/journald"
)

const (
	EventServiceState     = "ServiceState"
	EventStatusSnapshot   = "StatusSnapshot"
	EventWhitelistApplied = "WhitelistApplied"
	EventWhitelistExpired = "WhitelistExpired"
	EventNextKnockUpdated = "NextKnockUpdated"
	EventKnockTriggered   = "KnockTriggered"
	EventError            = "Error"
)

const (
	ServiceStateStarted  = "started"
	ServiceStateStopping = "stopping"
	ServiceStateStopped  = "stopped"
)

const (
	TriggerSourceCLI      = "cli"
	TriggerSourceSchedule = "schedule"
	TriggerSourceExternal = "external"
)

const (
	ResultSuccess = "success"
	ResultFailure = "failure"
)

const (
	ErrorCodeIPLookup    = "ip_lookup_failed"
	ErrorCodeHealthCheck = "health_check_failed"
	ErrorCodeKnockFailed = "knock_failed"
)

type whitelistState struct {
	IP          string
	ExpiresUnix int64
	TTLSeconds  int
	Source      string
}

func (s *Service) emit(eventType, message string, priority journald.Priority, fields journald.Fields) {
	if err := journald.Emit(eventType, message, priority, fields); err != nil && s.Logger != nil {
		s.Logger.Printf("Failed to emit journald event %s: %v", eventType, err)
	}
}

func (s *Service) emitServiceState(state string) {
	message := fmt.Sprintf("Service state: %s", state)
	fields := journald.Fields{
		"KNOCKER_SERVICE_STATE": state,
	}
	if s.version != "" {
		fields["KNOCKER_VERSION"] = s.version
	}
	priority := journald.PriInfo
	if state == ServiceStateStopping {
		priority = journald.PriNotice
	}
	s.emit(EventServiceState, message, priority, fields)
}

func (s *Service) emitStatusSnapshot() {
	fields := journald.Fields{}
	if s.currentWhitelist != nil {
		if s.currentWhitelist.IP != "" {
			fields["KNOCKER_WHITELIST_IP"] = s.currentWhitelist.IP
		}
		if s.currentWhitelist.ExpiresUnix > 0 {
			fields["KNOCKER_EXPIRES_UNIX"] = strconv.FormatInt(s.currentWhitelist.ExpiresUnix, 10)
		}
		if s.currentWhitelist.TTLSeconds > 0 {
			fields["KNOCKER_TTL_SEC"] = strconv.Itoa(s.currentWhitelist.TTLSeconds)
		}
	}
	if s.nextKnockUnix > 0 {
		fields["KNOCKER_NEXT_AT_UNIX"] = strconv.FormatInt(s.nextKnockUnix, 10)
	}
	s.emit(EventStatusSnapshot, "Status snapshot", journald.PriInfo, fields)
}

func (s *Service) emitWhitelistApplied(ip string, ttlSeconds int, expiresUnix int64, source string) {
	fields := journald.Fields{}
	if ip != "" {
		fields["KNOCKER_WHITELIST_IP"] = ip
	}
	if ttlSeconds > 0 {
		fields["KNOCKER_TTL_SEC"] = strconv.Itoa(ttlSeconds)
	}
	if expiresUnix > 0 {
		fields["KNOCKER_EXPIRES_UNIX"] = strconv.FormatInt(expiresUnix, 10)
	}
	if source != "" {
		fields["KNOCKER_SOURCE"] = source
	}

	var message string
	if ip == "" {
		message = "Whitelist updated"
	} else if ttlSeconds > 0 && expiresUnix > 0 {
		exp := time.Unix(expiresUnix, 0).UTC().Format(time.RFC3339)
		message = fmt.Sprintf("Whitelisted %s for %ds (expires at %s)", ip, ttlSeconds, exp)
	} else {
		message = fmt.Sprintf("Whitelisted %s", ip)
	}

	s.emit(EventWhitelistApplied, message, journald.PriInfo, fields)
}

func (s *Service) emitWhitelistExpired(ip string, expiredUnix int64) {
	fields := journald.Fields{}
	if ip != "" {
		fields["KNOCKER_WHITELIST_IP"] = ip
	}
	if expiredUnix > 0 {
		fields["KNOCKER_EXPIRED_UNIX"] = strconv.FormatInt(expiredUnix, 10)
	}

	message := "Whitelist expired"
	if ip != "" && expiredUnix > 0 {
		message = fmt.Sprintf("Whitelist expired for %s at %s", ip, time.Unix(expiredUnix, 0).UTC().Format(time.RFC3339))
	} else if ip != "" {
		message = fmt.Sprintf("Whitelist expired for %s", ip)
	}

	s.emit(EventWhitelistExpired, message, journald.PriNotice, fields)
}

func (s *Service) emitNextKnockUpdated(next time.Time) {
	fields := journald.Fields{}
	var message string

	if next.IsZero() {
		fields["KNOCKER_NEXT_AT_UNIX"] = "0"
		message = "Next knock cleared"
	} else {
		unix := next.Unix()
		fields["KNOCKER_NEXT_AT_UNIX"] = strconv.FormatInt(unix, 10)
		message = fmt.Sprintf("Next knock at %s", next.UTC().Format(time.RFC3339))
	}

	s.emit(EventNextKnockUpdated, message, journald.PriInfo, fields)
}

func (s *Service) emitKnockTriggered(source, result, ip string) {
	fields := journald.Fields{
		"KNOCKER_TRIGGER_SOURCE": source,
		"KNOCKER_RESULT":         result,
	}
	if ip != "" {
		fields["KNOCKER_WHITELIST_IP"] = ip
	}

	priority := journald.PriInfo
	if result != ResultSuccess {
		priority = journald.PriErr
	}

	message := fmt.Sprintf("Knock triggered via %s: %s", source, result)
	s.emit(EventKnockTriggered, message, priority, fields)
}

func (s *Service) emitError(code, msg, context string) {
	fields := journald.Fields{
		"KNOCKER_ERROR_CODE": code,
		"KNOCKER_ERROR_MSG":  msg,
	}
	if context != "" {
		fields["KNOCKER_CONTEXT"] = context
	}

	s.emit(EventError, msg, journald.PriErr, fields)
}

func (s *Service) updateNextKnock(next time.Time) {
	var unix int64
	if !next.IsZero() {
		unix = next.Unix()
	}

	if s.nextKnockUnix == unix {
		return
	}

	s.nextKnockUnix = unix
	s.emitNextKnockUpdated(next)
}

func (s *Service) clearNextKnock() {
	s.updateNextKnock(time.Time{})
}
