package fps

import (
	"golang.org/x/sys/windows"

	"github.com/MinhChau9208/NightreignCompanion/internal/etw"
)

// Microsoft-Windows-DXGI. Present_Start (event 42) fires for every
// IDXGISwapChain::Present call — D3D10/11/12 games, including Nightreign (DX12).
var dxgiProvider = etw.Provider{
	GUID: windows.GUID{
		Data1: 0xCA11C036, Data2: 0x0102, Data3: 0x4A2D,
		Data4: [8]byte{0xA6, 0xAD, 0xF0, 0x3C, 0xFE, 0xD5, 0xD3, 0xC9},
	},
	Level:    5,
	MatchAny: ^uint64(0), // all keywords (Present is on the Analytic channel)
}

const presentStartID = 42

// EnablePresents subscribes s to DXGI Present events.
func EnablePresents(s *etw.Session) error { return s.Enable(dxgiProvider) }

// PresentEvent reports whether e is a Present and, if so, which process
// presented and when (FILETIME ticks).
func PresentEvent(e *etw.Event) (pid uint32, ts int64, ok bool) {
	if e.ID() != presentStartID || e.Provider() != dxgiProvider.GUID {
		return 0, 0, false
	}
	return e.ProcessID(), e.TimeStamp(), true
}

// StartSession starts an ETW session that only listens for Presents.
func StartSession(name string, onPresent func(pid uint32, ts int64)) (*etw.Session, error) {
	s, err := etw.StartSession(name, func(e *etw.Event) {
		if pid, ts, ok := PresentEvent(e); ok {
			onPresent(pid, ts)
		}
	})
	if err != nil {
		return nil, err
	}
	if err := EnablePresents(s); err != nil {
		s.Close()
		return nil, err
	}
	return s, nil
}
