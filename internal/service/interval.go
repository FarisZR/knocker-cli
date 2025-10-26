package service

import "time"

const (
	defaultInterval     = 5 * time.Minute
	minInterval         = 5 * time.Second
	defaultRestartDelay = 30 * time.Second
	minRestartDelay     = 5 * time.Second
)

// EffectiveInterval ensures the knock loop respects the configured interval while
// also keeping the whitelist fresh relative to the configured TTL. When a TTL is
// provided, the interval is shortened so that a knock is scheduled once roughly
// 90%% of the TTL has elapsed (leaving a 10%% buffer before expiry), but never
// longer than either the TTL or the configured interval.
func EffectiveInterval(configured time.Duration, ttlSeconds int) time.Duration {
	if configured <= 0 {
		configured = defaultInterval
	}

	if ttlSeconds <= 0 {
		return configured
	}

	ttlDuration := time.Duration(ttlSeconds) * time.Second
	if ttlDuration <= 0 {
		return configured
	}

	adjusted := ttlDuration - (ttlDuration / 10) // aim to knock with 10% TTL remaining
	if adjusted <= 0 {
		adjusted = ttlDuration
	}
	if adjusted < time.Second {
		adjusted = time.Second
	}
	if ttlDuration >= minInterval && adjusted < minInterval {
		adjusted = minInterval
	}
	if adjusted > ttlDuration {
		adjusted = ttlDuration
	}

	if adjusted < configured {
		return adjusted
	}

	return configured
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
