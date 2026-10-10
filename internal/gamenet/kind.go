package gamenet

import "net/netip"

// valveRanges are address blocks of Valve's network (AS32590), where
// Steam Datagram Relay servers live. Not exhaustive: a relay outside these
// shows up as "peer", which is still an honest label for "some host on the
// internet the game exchanges UDP with".
var valveRanges = []netip.Prefix{
	netip.MustParsePrefix("103.10.124.0/23"),
	netip.MustParsePrefix("103.28.54.0/23"),
	netip.MustParsePrefix("146.66.152.0/21"),
	netip.MustParsePrefix("155.133.224.0/19"),
	netip.MustParsePrefix("162.254.192.0/21"),
	netip.MustParsePrefix("185.25.180.0/22"),
	netip.MustParsePrefix("205.196.6.0/24"),
	netip.MustParsePrefix("208.64.200.0/22"),
}

func classify(proto string, a netip.Addr) string {
	if a.IsPrivate() || a.IsLinkLocalUnicast() {
		return KindLAN
	}
	for _, p := range valveRanges {
		if p.Contains(a) {
			return KindRelay
		}
	}
	if proto == "tcp" {
		return KindServer
	}
	return KindPeer
}
