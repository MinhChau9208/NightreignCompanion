package netmon

import (
	"context"
	"testing"
	"time"
)

func TestICMPLoopback(t *testing.T) {
	p, probe, err := NewPinger(context.Background(), "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	if probe.Method != "icmp" || probe.Address != "127.0.0.1" {
		t.Fatalf("probe = %+v", probe)
	}
	if _, err := p.Ping(context.Background(), time.Second); err != nil {
		t.Fatalf("ping loopback: %v", err)
	}
}

func TestDefaultGateway(t *testing.T) {
	ip, err := defaultGateway()
	if err != nil {
		t.Skipf("no default gateway on this machine: %v", err)
	}
	if ip.To4() == nil || ip.IsUnspecified() {
		t.Fatalf("gateway = %v", ip)
	}
	t.Logf("default gateway: %v", ip)
}
