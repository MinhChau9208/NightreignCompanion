package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/png"
	"log"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/MinhChau9208/NightreignCompanion/internal/boss"
	"github.com/MinhChau9208/NightreignCompanion/internal/capture"
	"github.com/MinhChau9208/NightreignCompanion/internal/fps"
	"github.com/MinhChau9208/NightreignCompanion/internal/ocr"
	"github.com/MinhChau9208/NightreignCompanion/internal/timer"
)

// Boss OCR reads the names above the boss health bars while the timer is in
// a boss phase, and fills in that night's boss (or the Nightlord). It only
// captures screen pixels of the game window; see CLAUDE.md on EAC.

const (
	ocrLang      = "en-US" // the boss names in the data pack are English
	ocrInterval  = time.Second
	ocrRetryInit = 30 * time.Second
	// ocrConfirm is how many captures in a row must agree on a boss.
	ocrConfirm = 2
)

// OCR states shown in the UI.
const (
	ocrOff      = "off"      // disabled in settings
	ocrIdle     = "idle"     // timer not in a boss phase, or that boss is known
	ocrScanning = "scanning" // looking for a boss name
	ocrNoGame   = "noGame"   // game process or window not found
	ocrError    = "error"
)

type OCRStatus struct {
	State string `json:"state"`
	Slot  int    `json:"slot"` // 1, 2 = night boss, 3 = Nightlord; 0 when idle
	Error string `json:"error"`
}

type bossScanner struct {
	mu      sync.Mutex
	eng     *ocr.Engine
	engErr  error
	engTime time.Time

	status OCRStatus
	// candidate is the boss seen in the last captures, and how many times.
	candidate string
	streak    int
}

// engine starts the OCR engine on first use; a failure is retried at most
// every ocrRetryInit.
func (a *App) engine() (*ocr.Engine, error) {
	s := &a.scan
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.eng != nil {
		return s.eng, nil
	}
	if s.engErr != nil && time.Since(s.engTime) < ocrRetryInit {
		return nil, s.engErr
	}
	s.eng, s.engErr = ocr.New(ocrLang)
	s.engTime = time.Now()
	if s.engErr != nil {
		log.Printf("ocr: %v", s.engErr)
	}
	return s.eng, s.engErr
}

func (a *App) setOCRStatus(st OCRStatus) {
	s := &a.scan
	s.mu.Lock()
	changed := s.status != st
	s.status = st
	if st.State != ocrScanning {
		s.candidate, s.streak = "", 0
	}
	s.mu.Unlock()
	if changed {
		runtime.EventsEmit(a.ctx, "ocr:status", st)
	}
}

// runOCR reads the screen once a second: boss names (when the timer is in
// a boss phase) and the banners that sync the timer.
func (a *App) runOCR() {
	tick := time.NewTicker(ocrInterval)
	defer tick.Stop()
	defer func() {
		a.scan.mu.Lock()
		if a.scan.eng != nil {
			a.scan.eng.Close()
		}
		a.scan.mu.Unlock()
	}()
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-tick.C:
		}
		a.setOCRStatus(a.scanOnce())
		a.setSyncStatus(a.scanBanner())
	}
}

// bossSlot is the boss phase the timer is running: 1 and 2 for the night
// bosses, 3 for the Nightlord, 0 otherwise.
func (a *App) bossSlot() int {
	c := &a.tmr
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.t == nil {
		return 0
	}
	s := c.t.Snapshot()
	if s.State != timer.Running || s.Kind != "boss" {
		return 0
	}
	slot := 1
	for _, p := range c.t.Profile().Phases[:s.Phase] {
		if p.Kind == "boss" {
			slot++
		}
	}
	return min(slot, 3)
}

func (a *App) scanOnce() OCRStatus {
	cfg := a.cfg.Get()
	if !cfg.OCR.Enabled {
		return OCRStatus{State: ocrOff}
	}
	slot := a.bossSlot()
	if slot == 0 {
		return OCRStatus{State: ocrIdle}
	}
	if e := a.expedition(); e.get(slot) != "" {
		return OCRStatus{State: ocrIdle, Slot: slot}
	}
	img, err := a.captureBossNames(cfg.FPS.Process, cfg.OCR.Region)
	if errors.Is(err, capture.ErrNoWindow) || errors.Is(err, errNoGame) {
		return OCRStatus{State: ocrNoGame, Slot: slot}
	}
	if err != nil {
		return OCRStatus{State: ocrError, Slot: slot, Error: err.Error()}
	}
	lines, err := a.readText(img)
	if err != nil {
		return OCRStatus{State: ocrError, Slot: slot, Error: err.Error()}
	}
	id, _ := boss.Match(lines, a.aliases(slot))

	s := &a.scan
	s.mu.Lock()
	if id != "" && id == s.candidate {
		s.streak++
	} else {
		s.candidate, s.streak = id, 1
	}
	confirmed := id != "" && s.streak >= ocrConfirm
	s.mu.Unlock()
	if confirmed {
		log.Printf("ocr: %s boss is %s", slotField(slot), id)
		a.detected(slot, id)
		return OCRStatus{State: ocrIdle, Slot: slot}
	}
	return OCRStatus{State: ocrScanning, Slot: slot}
}

var errNoGame = errors.New("game is not running")

// captureBossNames grabs the configured region of the game window.
func (a *App) captureBossNames(process string, region capture.Region) (*image.RGBA, error) {
	pid, err := fps.FindProcess(process)
	if err != nil {
		return nil, err
	}
	if pid == 0 {
		return nil, errNoGame
	}
	win, err := capture.Window(pid)
	if err != nil {
		return nil, err
	}
	return capture.Grab(region.In(win))
}

// readText runs OCR on a capture, scaling small HUD text up first.
func (a *App) readText(img image.Image) ([]string, error) {
	eng, err := a.engine()
	if err != nil {
		return nil, err
	}
	scale := 1
	if img.Bounds().Dy() < 400 {
		scale = 2
	}
	return eng.Recognize(ocr.Prepare(img, scale))
}

// aliases lists the on-screen names that can appear in a boss slot.
func (a *App) aliases(slot int) []boss.Alias {
	var out []boss.Alias
	for _, b := range a.data.Bosses {
		ok := b.Kind == "night_boss" && b.Night == slot || b.Kind == "nightlord" && slot == 3
		if !ok {
			continue
		}
		for _, n := range a.data.ScreenNames(b.ID) {
			out = append(out, boss.Alias{ID: b.ID, Name: n})
		}
	}
	return out
}

// ---- Bound methods ----

func (a *App) OCRStatus() OCRStatus {
	a.scan.mu.Lock()
	defer a.scan.mu.Unlock()
	return a.scan.status
}

// OCRTest is one capture for checking the region in Settings.
type OCRTest struct {
	Window string   `json:"window"` // game window size, e.g. 1920x1080
	Region string   `json:"region"` // captured size
	Image  string   `json:"image"`  // PNG data URL of the capture
	Lines  []string `json:"lines"`  // text read
	Match  string   `json:"match"`  // boss id, any night, or banner kind; "" if none
	Day    int      `json:"day"`    // Day banner: 1-3
	Score  float64  `json:"score"`
}

// TestOCR captures region now and reports what OCR reads there, matched
// against every night boss and Nightlord.
func (a *App) TestOCR(region capture.Region) (OCRTest, error) {
	res, lines, err := a.testCapture(region)
	if err != nil {
		return res, err
	}
	all := append(append(a.aliases(1), a.aliases(2)...), a.aliases(3)...)
	res.Match, res.Score = boss.Match(lines, all)
	return res, nil
}

// testCapture grabs region of the game window and reads it, for checking
// a region in Settings.
func (a *App) testCapture(region capture.Region) (OCRTest, []string, error) {
	if err := a.runsErr(); err != nil {
		return OCRTest{}, nil, err
	}
	if !region.Valid() {
		return OCRTest{}, nil, errors.New("OCR region must lie inside the game window")
	}
	pid, err := fps.FindProcess(a.cfg.Get().FPS.Process)
	if err != nil {
		return OCRTest{}, nil, err
	}
	if pid == 0 {
		return OCRTest{}, nil, errNoGame
	}
	win, err := capture.Window(pid)
	if err != nil {
		return OCRTest{}, nil, err
	}
	r := region.In(win)
	img, err := capture.Grab(r)
	if err != nil {
		return OCRTest{}, nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return OCRTest{}, nil, err
	}
	res := OCRTest{
		Window: fmt.Sprintf("%dx%d", win.Dx(), win.Dy()),
		Region: fmt.Sprintf("%dx%d", r.Dx(), r.Dy()),
		Image:  "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()),
	}
	if res.Lines, err = a.readText(img); err != nil {
		return res, nil, err
	}
	return res, res.Lines, nil
}
