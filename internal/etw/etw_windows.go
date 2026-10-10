// Package etw runs real-time ETW sessions: the OS event stream that tools
// like PresentMon read. Consuming ETW never touches the game process, so it
// is safe with Easy Anti-Cheat, but creating a session needs Administrator
// rights; that is why it runs in the elevated helper (docs/SCOPE.md §2.4).
package etw

import (
	"errors"
	"fmt"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/windows"
)

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
	processTraceModeRealTime     = 0x00000100
	processTraceModeEventRecord  = 0x10000000
	invalidProcessTraceHandle    = ^uint64(0)
	errorAlreadyExists           = 183
	errorAccessDenied            = 5
	errorInvalidHandle           = 6
	errorCtxClosePending         = 6730
	loggerNameBytes              = 1024
	eventTracePropertiesBaseSize = 120
)

// ErrNeedsAdmin is returned when the ETW session cannot be created because
// the process is not elevated.
var ErrNeedsAdmin = errors.New("ETW tracing needs Administrator rights")

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

// eventRecord mirrors EVENT_RECORD.
type eventRecord struct {
	EventHeader       eventHeader
	BufferContext     uint32
	ExtendedDataCount uint16
	UserDataLength    uint16
	ExtendedData      uintptr
	UserDataPtr       unsafe.Pointer
	UserContext       uintptr
}

// Event is one event as delivered to a Handler. It points into ETW's
// buffer and is only valid during the call.
type Event eventRecord

func (e *Event) Provider() windows.GUID { return e.EventHeader.ProviderID }
func (e *Event) ID() uint16             { return e.EventHeader.EventDescriptor.ID }

// ProcessID is the process that was running when the event was logged. For
// kernel providers this is often not the process the event is about; those
// carry the PID in their payload.
func (e *Event) ProcessID() uint32 { return e.EventHeader.ProcessID }

// TimeStamp is in FILETIME ticks (100 ns since 1601).
func (e *Event) TimeStamp() int64 { return e.EventHeader.TimeStamp }

// UserData is the event payload; copy anything kept past the call.
func (e *Event) UserData() []byte {
	if e.UserDataPtr == nil || e.UserDataLength == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(e.UserDataPtr), e.UserDataLength)
}

// Handler is called on the ProcessTrace thread for every event; it must be
// quick, since ETW drops events while the consumer falls behind.
type Handler func(*Event)

// ETW calls back on the ProcessTrace thread through a single C callback, so
// the handler lives in a package variable; only one session runs per process.
var handler atomic.Pointer[Handler]

var eventCallback = windows.NewCallback(func(r *eventRecord) uintptr {
	if h := handler.Load(); h != nil {
		(*h)((*Event)(r))
	}
	return 0
})

// Provider is an ETW provider to enable on a session.
type Provider struct {
	GUID     windows.GUID
	Level    uint8  // 5 = verbose
	MatchAny uint64 // keyword mask; all ones enables every keyword
}

// Session is a real-time ETW session.
type Session struct {
	name    *uint16
	props   []uint64 // backing store, 8-byte aligned
	session uint64
	trace   uint64
	closed  atomic.Bool
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

// StartSession creates (or takes over a leftover) session called name that
// delivers events to h once Process runs. Enable providers before Process.
func StartSession(name string, h Handler) (*Session, error) {
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

	handler.Store(&h)
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

// Enable subscribes the session to p.
func (s *Session) Enable(p Provider) error {
	r, _, _ := procEnableTraceEx2.Call(
		uintptr(s.session),
		uintptr(unsafe.Pointer(&p.GUID)),
		eventControlCodeEnable,
		uintptr(p.Level),
		uintptr(p.MatchAny),
		0, // MatchAllKeyword
		0, // Timeout: asynchronous
		0,
	)
	if r != 0 {
		return fmt.Errorf("EnableTraceEx2: %w", windows.Errno(r))
	}
	return nil
}

// Process delivers events until Close is called. It blocks.
func (s *Session) Process() error {
	if s.closed.Load() {
		return nil
	}
	r, _, _ := procProcessTrace.Call(uintptr(unsafe.Pointer(&s.trace)), 1, 0, 0)
	// Close may run before ProcessTrace starts; the trace handle is then
	// already closed, which is a normal shutdown, not an error.
	if r == errorInvalidHandle && s.closed.Load() {
		return nil
	}
	if r != 0 && r != errorCtxClosePending {
		return fmt.Errorf("ProcessTrace: %w", windows.Errno(r))
	}
	return nil
}

// Close stops the session; Process returns shortly after.
func (s *Session) Close() error {
	handler.Store(nil)
	s.closed.Store(true)
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
