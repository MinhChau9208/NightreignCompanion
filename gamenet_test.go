package main

import (
	"slices"
	"testing"

	"github.com/MinhChau9208/NightreignCompanion/internal/gamenet"
)

func TestPeerTargets(t *testing.T) {
	flows := []gamenet.Flow{
		{Proto: "udp", IP: "155.133.230.34"},
		{Proto: "tcp", IP: "203.0.113.1"},               // TCP: not pinged
		{Proto: "udp", IP: "155.133.230.34"},            // same IP, other port
		{Proto: "udp", IP: "2001:db8::1"},               // ICMP prober is IPv4-only
		{Proto: "udp", IP: "203.0.113.2", IdleMs: 9000}, // gone quiet
		{Proto: "udp", IP: "203.0.113.3"},
		{Proto: "udp", IP: "203.0.113.4"},
		{Proto: "udp", IP: "203.0.113.5"},
		{Proto: "udp", IP: "203.0.113.6"},
	}
	want := []string{"155.133.230.34", "203.0.113.3", "203.0.113.4", "203.0.113.5"}
	if got := peerTargets(flows); !slices.Equal(got, want) {
		t.Errorf("peerTargets = %v, want %v", got, want)
	}
}
