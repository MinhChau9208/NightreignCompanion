package main

import (
	"time"

	"golang.org/x/sys/windows"
)

var procBeep = windows.NewLazySystemDLL("kernel32.dll").NewProc("Beep")

// beep plays a tone on the default audio device; it blocks for ms.
func beep(freq, ms int) { procBeep.Call(uintptr(freq), uintptr(ms)) }

// warnBeep counts down: one beep at 60s left, two at 30s, three at 10s.
func warnBeep(mark int) {
	n, freq := 1, 660
	switch {
	case mark <= 10:
		n, freq = 3, 880
	case mark <= 30:
		n = 2
	}
	for i := 0; i < n; i++ {
		beep(freq, 120)
		time.Sleep(80 * time.Millisecond)
	}
}

// phaseBeep marks a phase that started on its own (e.g. the circle closing).
func phaseBeep() { beep(990, 450) }
