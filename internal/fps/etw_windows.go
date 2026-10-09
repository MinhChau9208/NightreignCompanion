package fps

import (
	"errors"
	"fmt"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Microsoft-Windows-DXGI. Present_Start (event 42) fires for every
// IDXGISwapChain::Present call — D3D10/11/12 games, including Nightreign (DX12).
var dxgiProvider = windows.GUID{
	Data1: 0xCA11C036, Data2: 0x0102, Data3: 0x4A2D,
	Data4: [8]byte{0xA6, 0xAD, 0xF0, 0x3C, 0xFE, 0xD5, 0xD3, 0xC9},
}

const presentStartID = 42

var (
	advapi32           = windows.NewLazySystemDLL("advapi32.dll")
	procStartTraceW    = advapi32.NewProc("StartTraceW")
	procControlTraceW  = advapi32.NewProc("ControlTraceW")
	procEnableTraceEx2 = advapi32.NewProc("EnableTraceEx2")
	procOpenTraceW     = advapi32.NewProc("OpenTraceW")
	procProcessTrace   = advapi32.NewProc("ProcessTrace")
	procCloseTrace     = advapi32.NewProc("CloseTrace")
)

const (
	wnodeFlagTracedGUID          = 0x00020000
	eventTraceRealTimeMode       = 0x00000100
	eventTraceControlStop        = 1
	eventControlCodeEnable       = 1
	traceLevelVerbose            = 5
	processTraceModeRealTime     = 0x00000100
	processTraceModeEventRecord  = 0x10000000
	invalidProcessTraceHandle    = ^uint64(0)
	errorAlreadyExists           = 183
	errorAccessDenied            = 5
	errorCtxClosePending         = 6730
	loggerNameBytes              = 1024
	eventTracePropertiesBaseSize = 120
)

// ErrNeedsAdmin is returned when the ETW session cannot be created because
// the process is not elevated.
var ErrNeedsAdmin = errors.New("FPS measurement needs Administrator rights")

// The structs below mirror the Win32 x64 layouts (sizes asserted in tests).

type wnodeHeader struct {
	BufferSize        uint32
	ProviderID        uint32
	HistoricalContext uint64
	TimeStamp         int64
	GUID              windows.GUID
	ClientContext     uint32
	Flags             uint32
}

type eventTraceProperties struct {
	Wnode               wnodeHeader
	BufferSize          uint32
	MinimumBuffers      uint32
	MaximumBuffers      uint32
	MaximumFileSize     uint32
	LogFileMode         uint32
	FlushTimer          uint32
	EnableFlags         uint32
	AgeLimit            int32
	NumberOfBuffers     uint32
	FreeBuffers         uint32
	EventsLost          uint32
	BuffersWritten      uint32
	LogBuffersLost      uint32
	RealTimeBuffersLost uint32
	LoggerThreadID      uintptr
	LogFileNameOffset   uint32
	LoggerNameOffset    uint32
}

type eventTraceHeader struct {
	Size           uint16
	FieldTypeFlags uint16
	Version        uint32
	ThreadID       uint32
	ProcessID      uint32
	TimeStamp      int64
	GUID           windows.GUID
	ProcessorTime  uint64
}

type eventTrace struct {
	Header           eventTraceHeader
	InstanceID       uint32
	ParentInstanceID uint32
	ParentGUID       windows.GUID
	MofData          uintptr
	MofLength        uint32
	ClientContext    uint32
}

type systemTime struct {
	Year, Month, DayOfWeek, Day, Hour, Minute, Second, Milliseconds uint16
}

type timeZoneInformation struct {
	Bias         int32
	StandardName [32]uint16
	StandardDate systemTime
	StandardBias int32
	DaylightName [32]uint16
	DaylightDate systemTime
	DaylightBias int32
}

type traceLogfileHeader struct {
	BufferSize         uint32
	Version            uint32
	ProviderVersion    uint32
	NumberOfProcessors uint32
	EndTime            int64
	TimerResolution    uint32
	MaximumFileSize    uint32
	LogFileMode        uint32
	BuffersWritten     uint32
	InstanceUnion      [16]byte
	LoggerName         *uint16
	LogFileName        *uint16
	TimeZone           timeZoneInformation
	BootTime           int64
	PerfFreq           int64
	StartTime          int64
	ReservedFlags      uint32
	BuffersLost        uint32
}

type eventTraceLogfile struct {
	LogFileName      *uint16
	LoggerName       *uint16
	CurrentTime      int64
	BuffersRead      uint32
	ProcessTraceMode uint32
	CurrentEvent     eventTrace
	LogfileHeader    traceLogfileHeader
	BufferCallback   uintptr
	BufferSize       uint32
	Filled           uint32
	EventsLost       uint32
	EventCallback    uintptr
	IsKernelTrace    uint32
	Context          uintptr
}

type eventDescriptor struct {
	ID      uint16
	Version uint8
	Channel uint8
	Level   uint8
	Opcode  uint8
	Task    uint16
	Keyword uint64
}

type eventHeader struct {
	Size            uint16
	HeaderType      uint16
	Flags           uint16
	EventProperty   uint16
	ThreadID        uint32
	ProcessID       uint32
	TimeStamp       int64 // FILETIME (100 ns) since we don't ask for raw timestamps
	ProviderID      windows.GUID
	EventDescriptor eventDescriptor
	ProcessorTime   uint64
	ActivityID      windows.GUID
}

// eventRecord is the prefix of EVENT_RECORD we read.
type eventRecord struct {
	EventHeader eventHeader
}

// ETW calls back on the ProcessTrace thread through a single C callback, so
// the handler lives in a package variable; only one session runs per process.
var presentHandler atomic.Pointer[func(pid uint32, ts int64)]

var eventCallback = windows.NewCallback(func(r *eventRecord) uintptr {
	h := &r.EventHeader
	if h.EventDescriptor.ID == presentStartID && h.ProviderID == dxgiProvider {
		if fn := presentHandler.Load(); fn != nil {
			(*fn)(h.ProcessID, h.TimeStamp)
		}
	}
	return 0
})

// Session is a real-time ETW session subscribed to DXGI Present events.
type Session struct {
	name    *uint16
	props   []uint64 // backing store, 8-byte aligned
	session uint64
	trace   uint64
}

func (s *Session) properties() *eventTraceProperties {
	return (*eventTraceProperties)(unsafe.Pointer(&s.props[0]))
}

func newProperties() []uint64 {
	buf := make([]uint64, (eventTracePropertiesBaseSize+loggerNameBytes)/8)
	p := (*eventTraceProperties)(unsafe.Pointer(&buf[0]))
	p.Wnode.BufferSize = uint32(len(buf) * 8)
	p.Wnode.Flags = wnodeFlagTracedGUID
	p.Wnode.ClientContext = 1 // QPC clock
	p.LogFileMode = eventTraceRealTimeMode
	p.FlushTimer = 1 // seconds; keeps delivery latency low
	p.LoggerNameOffset = eventTracePropertiesBaseSize
	return buf
}

// StartSession creates (or takes over a leftover) session called name and
// calls onPresent for every Present from any process.
func StartSession(name string, onPresent func(pid uint32, ts int64)) (*Session, error) {
	n, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, err
	}
	s := &Session{name: n, props: newProperties()}

	r, _, _ := procStartTraceW.Call(uintptr(unsafe.Pointer(&s.session)), uintptr(unsafe.Pointer(n)), uintptr(unsafe.Pointer(s.properties())))
	if r == errorAlreadyExists {
		// A previous helper was killed before it could clean up.
		stop := newProperties()
		procControlTraceW.Call(0, uintptr(unsafe.Pointer(n)), uintptr(unsafe.Pointer(&stop[0])), eventTraceControlStop)
		s.props = newProperties()
		r, _, _ = procStartTraceW.Call(uintptr(unsafe.Pointer(&s.session)), uintptr(unsafe.Pointer(n)), uintptr(unsafe.Pointer(s.properties())))
	}
	switch r {
	case 0:
	case errorAccessDenied:
		return nil, ErrNeedsAdmin
	default:
		return nil, fmt.Errorf("StartTrace: %w", windows.Errno(r))
	}

	r, _, _ = procEnableTraceEx2.Call(
		uintptr(s.session),
		uintptr(unsafe.Pointer(&dxgiProvider)),
		eventControlCodeEnable,
		traceLevelVerbose,
		uintptr(^uint64(0)), // MatchAnyKeyword: all keywords (Present is on the Analytic channel)
		0,                   // MatchAllKeyword
		0,                   // Timeout: asynchronous
		0,
	)
	if r != 0 {
		s.stop()
		return nil, fmt.Errorf("EnableTraceEx2: %w", windows.Errno(r))
	}

	presentHandler.Store(&onPresent)
	logfile := eventTraceLogfile{
		LoggerName:       n,
		ProcessTraceMode: processTraceModeRealTime | processTraceModeEventRecord,
		EventCallback:    eventCallback,
	}
	t, _, callErr := procOpenTraceW.Call(uintptr(unsafe.Pointer(&logfile)))
	if uint64(t) == invalidProcessTraceHandle {
		s.stop()
		return nil, fmt.Errorf("OpenTrace: %w", callErr)
	}
	s.trace = uint64(t)
	return s, nil
}

// Process delivers events until Close is called. It blocks.
func (s *Session) Process() error {
	r, _, _ := procProcessTrace.Call(uintptr(unsafe.Pointer(&s.trace)), 1, 0, 0)
	if r != 0 && r != errorCtxClosePending {
		return fmt.Errorf("ProcessTrace: %w", windows.Errno(r))
	}
	return nil
}

// Close stops the session; Process returns shortly after.
func (s *Session) Close() error {
	presentHandler.Store(nil)
	if s.trace != 0 {
		procCloseTrace.Call(uintptr(s.trace))
	}
	return s.stop()
}

func (s *Session) stop() error {
	r, _, _ := procControlTraceW.Call(uintptr(s.session), 0, uintptr(unsafe.Pointer(s.properties())), eventTraceControlStop)
	if r != 0 {
		return fmt.Errorf("ControlTrace stop: %w", windows.Errno(r))
	}
	return nil
}
