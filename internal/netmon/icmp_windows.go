package netmon

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ICMP via iphlpapi's IcmpSendEcho: works for normal (non-Admin) users,
// unlike raw sockets.
var (
	iphlpapi           = windows.NewLazySystemDLL("iphlpapi.dll")
	procIcmpCreate     = iphlpapi.NewProc("IcmpCreateFile")
	procIcmpClose      = iphlpapi.NewProc("IcmpCloseHandle")
	procIcmpSendEcho   = iphlpapi.NewProc("IcmpSendEcho")
	procGetBestRoute   = iphlpapi.NewProc("GetBestRoute")
	errICMPUnavailable = errors.New("icmp: IcmpCreateFile failed")
)

const (
	ipSuccess     = 0
	ipReqTimedOut = 11010
)

// icmpEchoReply mirrors ICMP_ECHO_REPLY (64-bit layout).
type icmpEchoReply struct {
	Address       uint32
	Status        uint32
	RoundTripTime uint32
	DataSize      uint16
	Reserved      uint16
	Data          uintptr
	Options       struct {
		TTL, TOS, Flags, OptionsSize uint8
		OptionsData                  uintptr
	}
}

var payload = []byte("nightreign-companion")

type icmpPinger struct {
	handle uintptr
	dest   uint32
}

func newICMPPinger(ip net.IP) (Pinger, error) {
	h, _, _ := procIcmpCreate.Call()
	if h == uintptr(windows.InvalidHandle) || h == 0 {
		return nil, errICMPUnavailable
	}
	return &icmpPinger{handle: h, dest: binary.LittleEndian.Uint32(ip.To4())}, nil
}

func (p *icmpPinger) Ping(ctx context.Context, timeout time.Duration) (time.Duration, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	// Room for the reply struct, the echoed payload and an ICMP error message.
	buf := make([]byte, int(unsafe.Sizeof(icmpEchoReply{}))+len(payload)+64)
	n, _, callErr := procIcmpSendEcho.Call(
		p.handle,
		uintptr(p.dest),
		uintptr(unsafe.Pointer(&payload[0])),
		uintptr(len(payload)),
		0,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
		uintptr(timeout.Milliseconds()),
	)
	if n == 0 {
		if errno, ok := callErr.(windows.Errno); ok && errno == ipReqTimedOut {
			return 0, ErrTimeout
		}
		return 0, fmt.Errorf("icmp: %w", callErr)
	}
	reply := (*icmpEchoReply)(unsafe.Pointer(&buf[0]))
	switch reply.Status {
	case ipSuccess:
		return time.Duration(reply.RoundTripTime) * time.Millisecond, nil
	case ipReqTimedOut:
		return 0, ErrTimeout
	default:
		// Unreachable, TTL expired, …: the packet did not make it.
		return 0, fmt.Errorf("%w (icmp status %d)", ErrTimeout, reply.Status)
	}
}

func (p *icmpPinger) Close() error {
	procIcmpClose.Call(p.handle)
	return nil
}

// mibIPForwardRow mirrors MIB_IPFORWARDROW.
type mibIPForwardRow struct {
	Dest, Mask, Policy, NextHop, IfIndex, Type, Proto, Age, NextHopAS uint32
	Metric1, Metric2, Metric3, Metric4, Metric5                       uint32
}

// defaultGateway returns the next hop the OS would use to reach the internet.
func defaultGateway() (net.IP, error) {
	var row mibIPForwardRow
	dest := binary.LittleEndian.Uint32(net.IPv4(1, 1, 1, 1).To4())
	r, _, _ := procGetBestRoute.Call(uintptr(dest), 0, uintptr(unsafe.Pointer(&row)))
	if r != 0 {
		return nil, fmt.Errorf("GetBestRoute: %w", windows.Errno(r))
	}
	if row.NextHop == 0 {
		return nil, errors.New("no default gateway")
	}
	ip := make(net.IP, 4)
	binary.LittleEndian.PutUint32(ip, row.NextHop)
	return ip, nil
}
