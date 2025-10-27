//go:build linux

package journald

import (
	"github.com/coreos/go-systemd/v22/journal"
)

func enabled() bool {
	return journal.Enabled()
}

func emit(message string, priority Priority, fields Fields) error {
	payload := make(map[string]string, len(fields))
	for k, v := range fields {
		payload[k] = v
	}

	return journal.Send(message, journal.Priority(priority), payload)
}
