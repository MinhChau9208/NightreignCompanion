package gamenet

import (
	"net"
	"net/netip"

	"golang.org/x/sys/windows"

	"github.com/MinhChau9208/NightreignCompanion/internal/etw"
)

// Microsoft-Windows-Kernel-Network {7DD42A49-5329-4832-8DFD-43D979153A88}.
var kernelNetwork = etw.Provider{
	GUID: windows.GUID{
		Data1: 0x7DD42A49, Data2: 0x5329, Data3: 0x4832,
		Data4: [8]byte{0x8D, 0xFD, 0x43, 0xD9, 0x79, 0x15, 0x3A, 0x88},
	},
	Level:    4,          // informational: all send/receive events
	MatchAny: ^uint64(0), // IPv4 + IPv6 keywords and the Analytic channel
}

// Enable subscribes s to TCP/UDP send and receive events of every process.
func Enable(s *etw.Session) error { return s.Enable(kernelNetwork) }

// ParseEvent decodes a send/receive event. The owning PID comes from the
// payload: the event header's PID is whoever was on the CPU, which for
// receives is usually not the game.
func ParseEvent(e *etw.Event) (Packet, bool) {
	if e.Provider() != kernelNetwork.GUID {
		return Packet{}, false
	}
	return parse(e.ID(), e.UserData(), e.TimeStamp())
}

// LocalAddrs lists this machine's interface addresses.
func LocalAddrs() ([]netip.Addr, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}
	out := make([]netip.Addr, 0, len(addrs))
	for _, a := range addrs {
		if n, ok := a.(*net.IPNet); ok {
			if ip, ok := netip.AddrFromSlice(n.IP); ok {
				out = append(out, ip.Unmap())
			}
		}
	}
	return out, nil
}
