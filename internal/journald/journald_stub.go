//go:build !linux

package journald

func enabled() bool {
	return false
}

func emit(message string, priority Priority, fields Fields) error {
	return nil
}
