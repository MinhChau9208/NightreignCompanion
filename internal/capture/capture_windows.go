package capture

import (
	"fmt"
	"image"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32                 = windows.NewLazySystemDLL("user32.dll")
	gdi32                  = windows.NewLazySystemDLL("gdi32.dll")
	procGetClientRect      = user32.NewProc("GetClientRect")
	procClientToScreen     = user32.NewProc("ClientToScreen")
	procIsWindowVisible    = user32.NewProc("IsWindowVisible")
	procIsIconic           = user32.NewProc("IsIconic")
	procGetForegroundWnd   = user32.NewProc("GetForegroundWindow")
	procGetDC              = user32.NewProc("GetDC")
	procReleaseDC          = user32.NewProc("ReleaseDC")
	procCreateCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	procCreateDIBSection   = gdi32.NewProc("CreateDIBSection")
	procSelectObject       = gdi32.NewProc("SelectObject")
	procBitBlt             = gdi32.NewProc("BitBlt")
	procDeleteObject       = gdi32.NewProc("DeleteObject")
	procDeleteDC           = gdi32.NewProc("DeleteDC")
)

const srcCopy = 0x00CC0020

type rect struct{ Left, Top, Right, Bottom int32 }

type point struct{ X, Y int32 }

type bitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

// Go allows only a limited number of callbacks per process, so the
// EnumWindows callback is created once and searches are serialised.
var (
	searchMu sync.Mutex
	search   struct {
		pid  uint32
		best image.Rectangle
	}
	enumCB = syscall.NewCallback(func(hwnd windows.HWND, _ uintptr) uintptr {
		var owner uint32
		windows.GetWindowThreadProcessId(hwnd, &owner)
		if owner != search.pid {
			return 1
		}
		if v, _, _ := procIsWindowVisible.Call(uintptr(hwnd)); v == 0 {
			return 1
		}
		if iconic, _, _ := procIsIconic.Call(uintptr(hwnd)); iconic != 0 {
			return 1
		}
		if r, ok := clientRect(hwnd); ok && area(r) > area(search.best) {
			search.best = r
		}
		return 1
	})
)

// Window returns the client area, in screen pixels, of the largest visible
// top-level window owned by pid.
func Window(pid uint32) (image.Rectangle, error) {
	searchMu.Lock()
	defer searchMu.Unlock()
	search.pid, search.best = pid, image.Rectangle{}
	if err := windows.EnumWindows(enumCB, nil); err != nil {
		return image.Rectangle{}, err
	}
	if search.best.Empty() {
		return image.Rectangle{}, ErrNoWindow
	}
	return search.best, nil
}

// Foreground reports whether the window in front belongs to pid, i.e. the
// screen shows the game rather than another app on top of it.
func Foreground(pid uint32) bool {
	hwnd, _, _ := procGetForegroundWnd.Call()
	if hwnd == 0 {
		return false
	}
	var owner uint32
	windows.GetWindowThreadProcessId(windows.HWND(hwnd), &owner)
	return owner == pid
}

func area(r image.Rectangle) int { return r.Dx() * r.Dy() }

func clientRect(hwnd windows.HWND) (image.Rectangle, bool) {
	var r rect
	if ok, _, _ := procGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&r))); ok == 0 {
		return image.Rectangle{}, false
	}
	var p point
	if ok, _, _ := procClientToScreen.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&p))); ok == 0 {
		return image.Rectangle{}, false
	}
	return image.Rect(int(p.X), int(p.Y), int(p.X+r.Right), int(p.Y+r.Bottom)), true
}

// Grab copies the screen pixels inside r.
func Grab(r image.Rectangle) (*image.RGBA, error) {
	w, h := r.Dx(), r.Dy()
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("empty capture region %v", r)
	}
	screen, _, _ := procGetDC.Call(0)
	if screen == 0 {
		return nil, fmt.Errorf("GetDC failed")
	}
	defer procReleaseDC.Call(0, screen)
	mem, _, _ := procCreateCompatibleDC.Call(screen)
	if mem == 0 {
		return nil, fmt.Errorf("CreateCompatibleDC failed")
	}
	defer procDeleteDC.Call(mem)

	bi := bitmapInfoHeader{Width: int32(w), Height: -int32(h), Planes: 1, BitCount: 32} // top-down BGRA
	bi.Size = uint32(unsafe.Sizeof(bi))
	var bits unsafe.Pointer
	bmp, _, _ := procCreateDIBSection.Call(mem, uintptr(unsafe.Pointer(&bi)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if bmp == 0 || bits == nil {
		return nil, fmt.Errorf("CreateDIBSection failed")
	}
	defer procDeleteObject.Call(bmp)
	old, _, _ := procSelectObject.Call(mem, bmp)
	defer procSelectObject.Call(mem, old)
	if ok, _, _ := procBitBlt.Call(mem, 0, 0, uintptr(w), uintptr(h), screen, uintptr(r.Min.X), uintptr(r.Min.Y), srcCopy); ok == 0 {
		return nil, fmt.Errorf("BitBlt failed")
	}

	src := unsafe.Slice((*byte)(bits), w*h*4)
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(src); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = src[i+2], src[i+1], src[i], 255
	}
	return img, nil
}
