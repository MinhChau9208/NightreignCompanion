// Package netmon measures latency, jitter and packet loss to a set of
// targets (M1 Connection Checker, docs/SCOPE.md §3).
package netmon

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// ErrTimeout marks a probe that got no reply in time; it counts as a lost packet.
var ErrTimeout = errors.New("timeout")

// Pinger sends one probe and returns the round-trip time.
type Pinger interface {
	Ping(ctx context.Context, timeout time.Duration) (time.Duration, error)
	Close() error
}

// Probe describes how a target is measured.
type Probe struct {
	Method  string `json:"method"`  // icmp | tcp
	Address string `json:"address"` // resolved IP or host:port
}

// GatewayTarget is a special target name for the default gateway (router),
// which separates "my Wi-Fi/LAN is bad" from "my ISP/route is bad".
const GatewayTarget = "gateway"

// ParseTarget checks a user-entered target: "gateway", a host/IPv4 for
// ICMP, or host:port for a TCP connect probe.
func ParseTarget(target string) (method, host, port string, err error) {
	t := strings.TrimSpace(target)
	switch {
	case t == "":
		return "", "", "", errors.New("empty target")
	case strings.ContainsAny(t, " \t/\\"):
		return "", "", "", fmt.Errorf("target %q: invalid characters", t)
	case t == GatewayTarget:
		return "icmp", t, "", nil
	}
	if h, p, err := net.SplitHostPort(t); err == nil {
		if h == "" || p == "" {
			return "", "", "", fmt.Errorf("target %q: need host:port", t)
		}
		return "tcp", h, p, nil
	}
	return "icmp", t, "", nil
}

// NewPinger builds the pinger for target, resolving names to an address.
func NewPinger(ctx context.Context, target string) (Pinger, Probe, error) {
	method, host, port, err := ParseTarget(target)
	if err != nil {
		return nil, Probe{}, err
	}
	if method == "tcp" {
		addr := net.JoinHostPort(host, port)
		return tcpPinger{addr: addr}, Probe{Method: "tcp", Address: addr}, nil
	}

	var ip net.IP
	if host == GatewayTarget {
		if ip, err = defaultGateway(); err != nil {
			return nil, Probe{}, fmt.Errorf("default gateway: %w", err)
		}
	} else if ip, err = resolveIPv4(ctx, host); err != nil {
		return nil, Probe{}, err
	}
	p, err := newICMPPinger(ip)
	if err != nil {
		return nil, Probe{}, err
	}
	return p, Probe{Method: "icmp", Address: ip.String()}, nil
}

func resolveIPv4(ctx context.Context, host string) (net.IP, error) {
	if ip := net.ParseIP(host); ip != nil {
		if ip4 := ip.To4(); ip4 != nil {
			return ip4, nil
		}
		return nil, fmt.Errorf("%s: only IPv4 is supported for ICMP", host)
	}
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip4", host)
	if err != nil {
		return nil, err
	}
	return ips[0].To4(), nil
}

// tcpPinger measures TCP connect time; useful when ICMP is blocked.
type tcpPinger struct{ addr string }

func (p tcpPinger) Ping(ctx context.Context, timeout time.Duration) (time.Duration, error) {
	d := net.Dialer{Timeout: timeout}
	start := time.Now()
	c, err := d.DialContext(ctx, "tcp", p.addr)
	if err != nil {
		var ne net.Error
		if errors.As(err, &ne) && ne.Timeout() {
			return 0, ErrTimeout
		}
		return 0, err
	}
	rtt := time.Since(start)
	c.Close()
	return rtt, nil
}

func (tcpPinger) Close() error { return nil }
