package hotkey

import (
	"fmt"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32                 = windows.NewLazySystemDLL("user32.dll")
	procRegisterHotKey     = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey   = user32.NewProc("UnregisterHotKey")
	procGetMessageW        = user32.NewProc("GetMessageW")
	procPeekMessageW       = user32.NewProc("PeekMessageW")
	procPostThreadMessageW = user32.NewProc("PostThreadMessageW")
)

const (
	wmQuit      = 0x0012
	wmHotkey    = 0x0312
	wmUser      = 0x0400
	pmNoRemove  = 0x0000
	modNoRepeat = 0x4000 // holding the key down fires once
)

type msg struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      struct{ x, y int32 }
	private uint32
}

// Listener owns a thread with registered hotkeys.
type Listener struct {
	tid  uint32
	done chan struct{}
}

// Listen registers each binding with a non-empty Keys and calls on with the
// binding's Action whenever its hotkey is pressed. on runs on the hotkey
// thread, so it must not block. Bindings that fail to register (bad syntax,
// or the hotkey belongs to another app) are reported in errs; the others
// still work.
func Listen(binds []Binding, on func(action string)) (l *Listener, errs []error) {
	l = &Listener{done: make(chan struct{})}
	ready := make(chan []error)
	go func() {
		// Hotkey messages go to the thread that registered them.
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer close(l.done)

		// Make sure the thread has a message queue before Stop can post to it.
		var m msg
		procPeekMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, wmUser, wmUser, pmNoRemove)
		l.tid = windows.GetCurrentThreadId()

		var errs []error
		var ids []uintptr
		for i, b := range binds {
			if b.Keys == "" {
				continue
			}
			c, err := Parse(b.Keys)
			if err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", b.Action, err))
				continue
			}
			id := uintptr(i + 1)
			if r, _, err := procRegisterHotKey.Call(0, id, uintptr(c.Mods|modNoRepeat), uintptr(c.VK)); r == 0 {
				errs = append(errs, fmt.Errorf("%s: %s: %w", b.Action, c, err))
				continue
			}
			ids = append(ids, id)
		}
		defer func() {
			for _, id := range ids {
				procUnregisterHotKey.Call(0, id)
			}
		}()
		ready <- errs

		for {
			r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
			if int32(r) <= 0 { // WM_QUIT or error
				return
			}
			if m.message == wmHotkey && m.wParam >= 1 && int(m.wParam) <= len(binds) {
				on(binds[m.wParam-1].Action)
			}
		}
	}()
	return l, <-ready
}

// Stop unregisters the hotkeys and waits for the thread to exit.
func (l *Listener) Stop() {
	procPostThreadMessageW.Call(uintptr(l.tid), wmQuit, 0, 0)
	<-l.done
}
