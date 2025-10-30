package service

import "time"

const (
	defaultCheckInterval = 5 * time.Minute
	defaultKnockInterval = defaultCheckInterval
	defaultRestartDelay  = 30 * time.Second
	minRestartDelay      = 5 * time.Second
)

// DefaultCheckInterval exposes the built-in check interval used when the
// configuration omits a custom value or sets an invalid one.
func DefaultCheckInterval() time.Duration {
	return defaultCheckInterval
}

// NormalizeCheckInterval ensures the configured IP check cadence is usable.
func NormalizeCheckInterval(configured time.Duration) time.Duration {
	if configured <= 0 {
		return defaultCheckInterval
	}
	return configured
}

// KnockCadenceFromTTL derives the knock schedule based on the configured TTL.
// A 10%% buffer is maintained so the whitelist is refreshed before expiry. The
// cadence never exceeds the TTL itself and bottoms out at one second to avoid
// zero-duration tickers.
func KnockCadenceFromTTL(ttlSeconds int) time.Duration {
	if ttlSeconds <= 0 {
		return defaultKnockInterval
	}

	ttlDuration := time.Duration(ttlSeconds) * time.Second
	if ttlDuration <= 0 {
		return defaultKnockInterval
	}

	adjusted := ttlDuration - (ttlDuration / 10) // aim to knock with 10% TTL remaining
	if adjusted <= 0 {
		adjusted = ttlDuration
	}
	if adjusted < time.Second {
		adjusted = time.Second
	}
	if adjusted > ttlDuration {
		adjusted = ttlDuration
	}

	return adjusted
}

// RestartDelay derives a reasonable restart back-off for managed services based on the TTL.
// The delay is capped and floored to keep restart behaviour practical across platforms.
func RestartDelay(ttlSeconds int) time.Duration {
	if ttlSeconds <= 0 {
		return defaultRestartDelay
	}

	ttlDuration := time.Duration(ttlSeconds) * time.Second
	candidate := ttlDuration / 4
	if candidate < minRestartDelay {
		candidate = minRestartDelay
	}
	if candidate > defaultRestartDelay {
		candidate = defaultRestartDelay
	}

	return candidate
}
