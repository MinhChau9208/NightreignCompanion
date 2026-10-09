package fps

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unsafe"
)

// Win32 x64 struct sizes; a mismatch would corrupt memory in the syscalls.
func TestStructLayouts(t *testing.T) {
	cases := []struct {
		name      string
		got, want uintptr
	}{
		{"WNODE_HEADER", unsafe.Sizeof(wnodeHeader{}), 48},
		{"EVENT_TRACE_PROPERTIES", unsafe.Sizeof(eventTraceProperties{}), eventTracePropertiesBaseSize},
		{"EVENT_TRACE_HEADER", unsafe.Sizeof(eventTraceHeader{}), 48},
		{"EVENT_TRACE", unsafe.Sizeof(eventTrace{}), 88},
		{"TIME_ZONE_INFORMATION", unsafe.Sizeof(timeZoneInformation{}), 172},
		{"TRACE_LOGFILE_HEADER", unsafe.Sizeof(traceLogfileHeader{}), 280},
		{"EVENT_TRACE_LOGFILEW", unsafe.Sizeof(eventTraceLogfile{}), 448},
		{"EVENT_DESCRIPTOR", unsafe.Sizeof(eventDescriptor{}), 16},
		{"EVENT_HEADER", unsafe.Sizeof(eventHeader{}), 80},
		{"SHELLEXECUTEINFOW", unsafe.Sizeof(shellExecuteInfo{}), 112},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s: size %d, want %d", c.name, c.got, c.want)
		}
	}
	if off := unsafe.Offsetof(eventHeader{}.EventDescriptor); off != 40 {
		t.Errorf("EVENT_HEADER.EventDescriptor offset %d, want 40", off)
	}
	if off := unsafe.Offsetof(eventTraceLogfile{}.EventCallback); off != 424 {
		t.Errorf("EVENT_TRACE_LOGFILEW.EventRecordCallback offset %d, want 424", off)
	}
}

// Without elevation the session must fail cleanly with ErrNeedsAdmin;
// with elevation it must start and stop.
func TestStartSession(t *testing.T) {
	s, err := StartSession("NightreignCompanion-Test", func(uint32, int64) {})
	if errors.Is(err, ErrNeedsAdmin) {
		t.Skip("not elevated: got ErrNeedsAdmin as expected")
	}
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- s.Process() }()
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestFindProcess(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	pid, err := FindProcess(strings.ToUpper(filepath.Base(exe)))
	if err != nil {
		t.Fatal(err)
	}
	if pid == 0 {
		t.Fatal("did not find the test binary itself")
	}
	if pid, _ := FindProcess("definitely-not-running-xyz.exe"); pid != 0 {
		t.Fatalf("found pid %d for a missing process", pid)
	}
}
