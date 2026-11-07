//go:build linux

package service

import (
	"context"
	"io"
	"log"
	"testing"
	"time"
)

func TestNetworkListenerCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	changes, err := watchNetworkChanges(ctx, log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	timeout := time.NewTimer(time.Second)
	defer timeout.Stop()
	for {
		select {
		case _, ok := <-changes:
			if !ok {
				return
			}
		case <-timeout.C:
			t.Fatal("network listener did not close on cancellation")
		}
	}
}
