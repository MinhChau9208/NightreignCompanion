package fps

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unsafe"
)

// SHELLEXECUTEINFOW x64 size; a mismatch would corrupt memory in the syscall.
func TestShellExecuteInfoLayout(t *testing.T) {
	if got := unsafe.Sizeof(shellExecuteInfo{}); got != 112 {
		t.Errorf("SHELLEXECUTEINFOW: size %d, want 112", got)
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
