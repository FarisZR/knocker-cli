//go:build !linux

package service

import (
	"context"
	"fmt"
	"log"
	"net"
	"slices"
	"strings"
	"time"
)

// Other platforms detect interface/address changes by taking local snapshots.
// Periodic server status checks also cover route and upstream public IP changes.
func watchNetworkChanges(ctx context.Context, logger *log.Logger) (<-chan struct{}, error) {
	snapshot := func() (string, error) {
		interfaces, err := net.Interfaces()
		if err != nil {
			return "", err
		}
		var entries []string
		for _, iface := range interfaces {
			entries = append(entries, fmt.Sprintf("%d:%s:%v", iface.Index, iface.Name, iface.Flags))
			addresses, err := iface.Addrs()
			if err != nil {
				return "", err
			}
			for _, address := range addresses {
				entries = append(entries, fmt.Sprintf("%d:%s", iface.Index, address))
			}
		}
		slices.Sort(entries)
		return strings.Join(entries, "\n"), nil
	}
	previous, err := snapshot()
	if err != nil {
		return nil, err
	}
	changes := make(chan struct{}, 1)
	go func() {
		defer close(changes)
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				current, err := snapshot()
				if err != nil {
					logger.Printf("Network snapshot failed: %v", err)
					continue
				}
				if current != previous {
					previous = current
					select {
					case changes <- struct{}{}:
					default:
					}
				}
			}
		}
	}()
	return changes, nil
}
