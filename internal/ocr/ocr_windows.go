package ocr

import (
	"errors"
	"fmt"
	"image"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// WinRT is reached through raw vtables: every interface starts with
// IUnknown (0-2) and IInspectable (3-5), so methods start at index 6.

var (
	combase                    = windows.NewLazySystemDLL("combase.dll")
	procRoInitialize           = combase.NewProc("RoInitialize")
	procRoGetActivationFactory = combase.NewProc("RoGetActivationFactory")
	procWindowsCreateString    = combase.NewProc("WindowsCreateString")
	procWindowsDeleteString    = combase.NewProc("WindowsDeleteString")
	procWindowsGetStringRaw    = combase.NewProc("WindowsGetStringRawBuffer")
)

var (
	iidLanguageFactory      = mustGUID("{9B0252AC-0C27-44F8-B792-9793FB66C63E}")
	iidOcrEngineStatics     = mustGUID("{5BFFA85A-3384-3540-9940-699120D428A8}")
	iidCryptoBufferStatics  = mustGUID("{320B7E22-3CB0-4CDF-8663-1D28910065EB}")
	iidSoftwareBitmapStatic = mustGUID("{DF0385DB-672F-4A9D-806E-C2442F343E86}")
	iidAsyncInfo            = mustGUID("{00000036-0000-0000-C000-000000000046}")
)

const (
	pixelFormatGray8 = 62
	asyncStarted     = 0
	asyncCompleted   = 1
	roInitMulti      = 1
	rpcEChangedMode  = 0x80010106
	maxImageSide     = 10000 // OcrEngine.MaxImageDimension
)

func mustGUID(s string) windows.GUID {
	g, err := windows.GUIDFromString(s)
	if err != nil {
		panic(err)
	}
	return g
}

type hresultError struct {
	what string
	hr   uint32
}

func (e hresultError) Error() string { return fmt.Sprintf("%s: HRESULT 0x%08X", e.what, e.hr) }

// call invokes method idx of the COM object obj.
func call(what string, obj unsafe.Pointer, idx int, args ...uintptr) error {
	vtbl := *(*unsafe.Pointer)(obj)
	fn := *(*uintptr)(unsafe.Add(vtbl, idx*int(unsafe.Sizeof(uintptr(0)))))
	hr, _, _ := syscall.SyscallN(fn, append([]uintptr{uintptr(obj)}, args...)...)
	if int32(hr) < 0 {
		return hresultError{what, uint32(hr)}
	}
	return nil
}

func release(obj unsafe.Pointer) {
	if obj != nil {
		call("Release", obj, 2)
	}
}

func newHString(s string) (unsafe.Pointer, error) {
	u, err := windows.UTF16FromString(s)
	if err != nil {
		return nil, err
	}
	var h unsafe.Pointer
	hr, _, _ := procWindowsCreateString.Call(uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)-1), uintptr(unsafe.Pointer(&h)))
	if int32(hr) < 0 {
		return nil, hresultError{"WindowsCreateString", uint32(hr)}
	}
	return h, nil
}

func deleteHString(h unsafe.Pointer) {
	if h != nil {
		procWindowsDeleteString.Call(uintptr(h))
	}
}

// takeHString converts h to Go and frees it.
func takeHString(h unsafe.Pointer) string {
	if h == nil {
		return ""
	}
	defer deleteHString(h)
	var n uint32
	p, _, _ := procWindowsGetStringRaw.Call(uintptr(h), uintptr(unsafe.Pointer(&n)))
	if p == 0 || n == 0 {
		return ""
	}
	// Reinterpret the returned address without a uintptr→Pointer cast.
	str := *(**uint16)(unsafe.Pointer(&p))
	return windows.UTF16ToString(unsafe.Slice(str, n))
}

func factory(class string, iid *windows.GUID) (unsafe.Pointer, error) {
	h, err := newHString(class)
	if err != nil {
		return nil, err
	}
	defer deleteHString(h)
	var f unsafe.Pointer
	hr, _, _ := procRoGetActivationFactory.Call(uintptr(h), uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(&f)))
	if int32(hr) < 0 {
		return nil, hresultError{"RoGetActivationFactory " + class, uint32(hr)}
	}
	return f, nil
}

// Engine recognises text in one language. WinRT objects stay on a single
// OS thread, so every call is handed to that thread.
type Engine struct {
	reqs chan request
	done chan struct{}
}

type request struct {
	img   *image.Gray
	reply chan result
}

type result struct {
	lines []string
	err   error
}

// worker holds the WinRT objects owned by the engine thread.
type worker struct {
	engine, crypto, bitmaps unsafe.Pointer
}

// New starts an OCR engine for a BCP-47 language tag such as "en-US".
// It fails when that language's OCR pack is not installed.
func New(lang string) (*Engine, error) {
	e := &Engine{reqs: make(chan request), done: make(chan struct{})}
	ready := make(chan error, 1)
	go e.run(lang, ready)
	if err := <-ready; err != nil {
		return nil, err
	}
	return e, nil
}

func (e *Engine) run(lang string, ready chan<- error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(e.done)

	hr, _, _ := procRoInitialize.Call(roInitMulti)
	if int32(hr) < 0 && uint32(hr) != rpcEChangedMode {
		ready <- hresultError{"RoInitialize", uint32(hr)}
		return
	}
	w, err := open(lang)
	if err != nil {
		ready <- err
		return
	}
	defer w.close()
	ready <- nil
	for r := range e.reqs {
		lines, err := w.recognize(r.img)
		r.reply <- result{lines, err}
	}
}

func open(lang string) (w *worker, err error) {
	w = &worker{}
	defer func() {
		if err != nil {
			w.close()
		}
	}()
	langs, err := factory("Windows.Globalization.Language", &iidLanguageFactory)
	if err != nil {
		return nil, err
	}
	defer release(langs)
	tag, err := newHString(lang)
	if err != nil {
		return nil, err
	}
	defer deleteHString(tag)
	var language unsafe.Pointer
	if err := call("CreateLanguage", langs, 6, uintptr(tag), uintptr(unsafe.Pointer(&language))); err != nil {
		return nil, err
	}
	defer release(language)

	statics, err := factory("Windows.Media.Ocr.OcrEngine", &iidOcrEngineStatics)
	if err != nil {
		return nil, err
	}
	defer release(statics)
	var ok bool
	if err := call("IsLanguageSupported", statics, 8, uintptr(language), uintptr(unsafe.Pointer(&ok))); err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("Windows has no OCR pack for %s (add the language in Settings › Time & language)", lang)
	}
	if err := call("TryCreateFromLanguage", statics, 9, uintptr(language), uintptr(unsafe.Pointer(&w.engine))); err != nil {
		return nil, err
	}
	if w.engine == nil {
		return nil, fmt.Errorf("could not create an OCR engine for %s", lang)
	}
	if w.crypto, err = factory("Windows.Security.Cryptography.CryptographicBuffer", &iidCryptoBufferStatics); err != nil {
		return nil, err
	}
	if w.bitmaps, err = factory("Windows.Graphics.Imaging.SoftwareBitmap", &iidSoftwareBitmapStatic); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *worker) close() {
	release(w.engine)
	release(w.crypto)
	release(w.bitmaps)
}

func (w *worker) recognize(img *image.Gray) ([]string, error) {
	width, height := img.Rect.Dx(), img.Rect.Dy()
	if width == 0 || height == 0 {
		return nil, nil
	}
	if width > maxImageSide || height > maxImageSide {
		return nil, fmt.Errorf("image %dx%d is larger than %d", width, height, maxImageSide)
	}
	pix := img.Pix
	if img.Stride != width {
		pix = make([]byte, 0, width*height)
		for y := 0; y < height; y++ {
			pix = append(pix, img.Pix[y*img.Stride:y*img.Stride+width]...)
		}
	}

	var buf unsafe.Pointer
	if err := call("CreateFromByteArray", w.crypto, 9, uintptr(len(pix)), uintptr(unsafe.Pointer(&pix[0])), uintptr(unsafe.Pointer(&buf))); err != nil {
		return nil, err
	}
	defer release(buf)
	var bmp unsafe.Pointer
	if err := call("CreateCopyFromBuffer", w.bitmaps, 9, uintptr(buf), pixelFormatGray8,
		uintptr(width), uintptr(height), uintptr(unsafe.Pointer(&bmp))); err != nil {
		return nil, err
	}
	defer release(bmp)

	var op unsafe.Pointer
	if err := call("RecognizeAsync", w.engine, 6, uintptr(bmp), uintptr(unsafe.Pointer(&op))); err != nil {
		return nil, err
	}
	defer release(op)
	if err := wait(op); err != nil {
		return nil, err
	}
	var res unsafe.Pointer
	if err := call("GetResults", op, 8, uintptr(unsafe.Pointer(&res))); err != nil {
		return nil, err
	}
	defer release(res)
	return lines(res)
}

// wait polls an IAsyncOperation until it finishes. Recognition of a HUD
// strip takes tens of milliseconds, so polling is simpler than a callback.
func wait(op unsafe.Pointer) error {
	var info unsafe.Pointer
	if err := call("QueryInterface IAsyncInfo", op, 0, uintptr(unsafe.Pointer(&iidAsyncInfo)), uintptr(unsafe.Pointer(&info))); err != nil {
		return err
	}
	defer release(info)
	deadline := time.Now().Add(10 * time.Second)
	for {
		var status int32
		if err := call("get_Status", info, 7, uintptr(unsafe.Pointer(&status))); err != nil {
			return err
		}
		switch status {
		case asyncStarted:
			if time.Now().After(deadline) {
				call("Cancel", info, 9)
				return errors.New("OCR timed out")
			}
			time.Sleep(2 * time.Millisecond)
		case asyncCompleted:
			return nil
		default:
			var code uint32
			call("get_ErrorCode", info, 8, uintptr(unsafe.Pointer(&code)))
			return hresultError{"RecognizeAsync", code}
		}
	}
}

// lines reads OcrResult.Lines[i].Text.
func lines(res unsafe.Pointer) ([]string, error) {
	var view unsafe.Pointer
	if err := call("get_Lines", res, 6, uintptr(unsafe.Pointer(&view))); err != nil {
		return nil, err
	}
	defer release(view)
	var n uint32
	if err := call("get_Size", view, 7, uintptr(unsafe.Pointer(&n))); err != nil {
		return nil, err
	}
	out := make([]string, 0, n)
	for i := uint32(0); i < n; i++ {
		var line unsafe.Pointer
		if err := call("GetAt", view, 6, uintptr(i), uintptr(unsafe.Pointer(&line))); err != nil {
			return nil, err
		}
		var h unsafe.Pointer
		err := call("get_Text", line, 7, uintptr(unsafe.Pointer(&h)))
		release(line)
		if err != nil {
			return nil, err
		}
		out = append(out, takeHString(h))
	}
	return out, nil
}

// Recognize returns the text lines found in img.
func (e *Engine) Recognize(img *image.Gray) ([]string, error) {
	reply := make(chan result, 1)
	select {
	case e.reqs <- request{img, reply}:
	case <-e.done:
		return nil, errors.New("OCR engine closed")
	}
	r := <-reply
	return r.lines, r.err
}

// Close stops the engine thread.
func (e *Engine) Close() {
	close(e.reqs)
	<-e.done
}
