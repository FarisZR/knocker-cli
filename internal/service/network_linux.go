//go:build linux

package service

import (
	"context"
	"errors"
	"log"

	"golang.org/x/sys/unix"
)

// A route netlink subscription is read-only and does not require root.
func watchNetworkChanges(ctx context.Context, logger *log.Logger) (<-chan struct{}, error) {
	fd, err := unix.Socket(unix.AF_NETLINK, unix.SOCK_RAW|unix.SOCK_CLOEXEC|unix.SOCK_NONBLOCK, unix.NETLINK_ROUTE)
	if err != nil {
		return nil, err
	}
	groups := uint32(unix.RTMGRP_LINK | unix.RTMGRP_IPV4_IFADDR | unix.RTMGRP_IPV6_IFADDR | unix.RTMGRP_IPV4_ROUTE | unix.RTMGRP_IPV6_ROUTE)
	if err := unix.Bind(fd, &unix.SockaddrNetlink{Family: unix.AF_NETLINK, Groups: groups}); err != nil {
		unix.Close(fd)
		return nil, err
	}
	changes := make(chan struct{}, 1)
	go func() {
		defer close(changes)
		defer unix.Close(fd)
		buf := make([]byte, 65536)
		for ctx.Err() == nil {
			// Bounded poll makes cancellation independent of further network events.
			fds := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
			n, err := unix.Poll(fds, 250)
			if errors.Is(err, unix.EINTR) {
				continue
			}
			if err != nil {
				logger.Printf("Network listener stopped: %v", err)
				return
			}
			if n == 0 {
				continue
			}
			_, _, err = unix.Recvfrom(fd, buf, 0)
			if errors.Is(err, unix.EAGAIN) || errors.Is(err, unix.EINTR) {
				continue
			}
			if err != nil && !errors.Is(err, unix.ENOBUFS) {
				logger.Printf("Network listener stopped: %v", err)
				return
			}
			// Every subscribed group describes a link, address, or route
			// change. Lost notifications (ENOBUFS) also require a status check;
			// the server decides whether a knock is necessary.
			select {
			case changes <- struct{}{}:
			default:
			}
		}
	}()
	return changes, nil
}
