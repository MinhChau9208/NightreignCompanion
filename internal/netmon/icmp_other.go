//go:build !windows

package netmon

import (
	"errors"
	"net"
)

// The app targets Windows; other platforms only need to compile (CLI, tests).
var errUnsupported = errors.New("icmp: only supported on Windows; use a host:port TCP target")

func newICMPPinger(net.IP) (Pinger, error) { return nil, errUnsupported }

func defaultGateway() (net.IP, error) { return nil, errUnsupported }
