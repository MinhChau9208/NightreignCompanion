package main

import (
	"slices"
	"testing"

	"github.com/MinhChau9208/NightreignCompanion/internal/gamenet"
	"github.com/MinhChau9208/NightreignCompanion/internal/netmon"
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
	flows[0].Kind = gamenet.KindRelay
	want := []string{"155.133.230.34", "155.133.230.1", "203.0.113.3", "203.0.113.4", "203.0.113.5"}
	if got := peerTargets(flows); !slices.Equal(got, want) {
		t.Errorf("peerTargets = %v, want %v", got, want)
	}
}

func TestAttachPingsFallsBackToSiteRouter(t *testing.T) {
	flows := []gamenet.Flow{
		{Proto: "udp", Kind: gamenet.KindRelay, IP: "103.28.54.185"},
		{Proto: "udp", Kind: gamenet.KindPeer, IP: "203.0.113.7"},
		{Proto: "tcp", Kind: gamenet.KindServer, IP: "44.224.9.80"},
	}
	pings := map[string]netmon.Stats{
		"103.28.54.185": {Target: "103.28.54.185", Sent: 10},
		"103.28.54.1":   {Target: "103.28.54.1", Sent: 10, Received: 10, AvgMs: 74},
		"203.0.113.7":   {Target: "203.0.113.7", Sent: 10},
		"44.224.9.80":   {Target: "44.224.9.80", Sent: 10, Received: 10},
	}
	attachPings(flows, pings)
	if p := flows[0].Ping; p == nil || p.AvgMs != 74 || flows[0].PingAddr != "103.28.54.1" {
		t.Errorf("relay ping = %+v via %q", p, flows[0].PingAddr)
	}
	if p := flows[1].Ping; p == nil || p.Received != 0 || flows[1].PingAddr != "" {
		t.Errorf("silent peer should keep its own (unanswered) ping: %+v via %q", p, flows[1].PingAddr)
	}
	if flows[2].Ping != nil {
		t.Error("TCP flows are not pinged")
	}
}
