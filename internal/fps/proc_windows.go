package fps

import (
	"errors"
	"fmt"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// FindProcess returns the PID of the first running process whose image
// name matches exe (case-insensitive), or 0.
func FindProcess(exe string) (uint32, error) {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return 0, err
	}
	defer windows.CloseHandle(snap)

	var e windows.ProcessEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	for err = windows.Process32First(snap, &e); err == nil; err = windows.Process32Next(snap, &e) {
		if strings.EqualFold(windows.UTF16ToString(e.ExeFile[:]), exe) {
			return e.ProcessID, nil
		}
	}
	if errors.Is(err, windows.ERROR_NO_MORE_FILES) {
		return 0, nil
	}
	return 0, err
}

var (
	shell32             = windows.NewLazySystemDLL("shell32.dll")
	procShellExecuteExW = shell32.NewProc("ShellExecuteExW")
)

// shellExecuteInfo mirrors SHELLEXECUTEINFOW (x64).
type shellExecuteInfo struct {
	Size         uint32
	Mask         uint32
	Hwnd         uintptr
	Verb         *uint16
	File         *uint16
	Parameters   *uint16
	Directory    *uint16
	Show         int32
	InstApp      uintptr
	IDList       uintptr
	Class        *uint16
	KeyClass     uintptr
	HotKey       uint32
	IconOrMonitr uintptr
	Process      windows.Handle
}

const (
	seeMaskNoCloseProcess = 0x00000040
	seeMaskNoAsync        = 0x00000100
	swHide                = 0
)

// ErrCancelled means the user declined the UAC prompt.
var ErrCancelled = errors.New("administrator permission was declined")

// Proc is a process started by LaunchElevated.
type Proc struct{ h windows.Handle }

// LaunchElevated runs exe with args through the UAC "runas" verb.
func LaunchElevated(exe, args string) (*Proc, error) {
	verb, _ := windows.UTF16PtrFromString("runas")
	file, err := windows.UTF16PtrFromString(exe)
	if err != nil {
		return nil, err
	}
	params, err := windows.UTF16PtrFromString(args)
	if err != nil {
		return nil, err
	}
	info := shellExecuteInfo{
		Mask:       seeMaskNoCloseProcess | seeMaskNoAsync,
		Verb:       verb,
		File:       file,
		Parameters: params,
		Show:       swHide,
	}
	info.Size = uint32(unsafe.Sizeof(info))
	ok, _, callErr := procShellExecuteExW.Call(uintptr(unsafe.Pointer(&info)))
	if ok == 0 {
		if errors.Is(callErr, windows.ERROR_CANCELLED) {
			return nil, ErrCancelled
		}
		return nil, fmt.Errorf("ShellExecuteEx: %w", callErr)
	}
	if info.Process == 0 {
		return nil, errors.New("ShellExecuteEx returned no process handle")
	}
	return &Proc{h: info.Process}, nil
}

// Wait blocks until the process exits and releases its handle.
func (p *Proc) Wait() {
	windows.WaitForSingleObject(p.h, windows.INFINITE)
	windows.CloseHandle(p.h)
}
