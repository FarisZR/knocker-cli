package journald

import (
	"errors"
	"fmt"
	"math"
	"sync/atomic"
	"syscall"
)

// Priority mirrors journald priorities so we can defer to the platform-specific
// implementation without importing go-systemd on every platform.
type Priority int

// Supported syslog priorities.
const (
	PriEmerg  Priority = 0
	PriAlert  Priority = 1
	PriCrit   Priority = 2
	PriErr    Priority = 3
	PriWarn   Priority = 4
	PriNotice Priority = 5
	PriInfo   Priority = 6
	PriDebug  Priority = 7
)

// Fields represents the additional structured metadata attached to a journald
// entry.
type Fields map[string]string

const (
	// SchemaVersion captures the stable contract version for structured Knocker
	// journald entries.
	SchemaVersion     = "1"
	defaultIdentifier = "knocker"
)

// Enabled reports whether structured journald logging is available on the
// current platform/runtime.
func Enabled() bool {
	return enabled()
}

var journaldDisabled atomic.Bool

// Emit writes a journald entry that includes both a friendly human-readable
// message and structured fields that automation (like the GNOME extension) can
// consume. When journald is unavailable, Emit becomes a no-op.
func Emit(eventType, message string, priority Priority, fields Fields) error {
	if journaldDisabled.Load() {
		return nil
	}

	if len(fields) > math.MaxInt-3 {
		return fmt.Errorf("too many fields in journald entry: %d > %d", len(fields), math.MaxInt-3)
	}

	payload := make(Fields, len(fields)+3)
	for k, v := range fields {
		payload[k] = v
	}

	if eventType != "" {
		payload["KNOCKER_EVENT"] = eventType
	}
	if _, exists := payload["SYSLOG_IDENTIFIER"]; !exists {
		payload["SYSLOG_IDENTIFIER"] = defaultIdentifier
	}
	if _, exists := payload["KNOCKER_SCHEMA_VERSION"]; !exists {
		payload["KNOCKER_SCHEMA_VERSION"] = SchemaVersion
	}

	if err := emit(message, priority, payload); err != nil {
		if disableOn(err) {
			journaldDisabled.Store(true)
			return nil
		}
		return err
	}

	return nil
}

func disableOn(err error) bool {
	return errors.Is(err, syscall.ENOENT) ||
		errors.Is(err, syscall.ENOTDIR) ||
		errors.Is(err, syscall.ENOTCONN) ||
		errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.EPERM)
}
