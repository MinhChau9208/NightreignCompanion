package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/MinhChau9208/NightreignCompanion/internal/config"
	"github.com/MinhChau9208/NightreignCompanion/internal/gamedata"
	"github.com/MinhChau9208/NightreignCompanion/internal/hotkey"
	"github.com/MinhChau9208/NightreignCompanion/internal/timer"
)

// alertMarks are the seconds left in a timed phase at which the timer
// beeps (when sound is on).
var alertMarks = []int{60, 30, 10}

// timerCtl owns the cycle timer and the global hotkeys that drive it.
type timerCtl struct {
	mu   sync.Mutex
	t    *timer.Timer
	last timer.Snapshot // last published
	pub  time.Time

	keys      *hotkey.Listener
	keysCfg   config.Hotkeys
	keyErrs   []string
	suspended bool
}

// Hotkey actions, named after the config.Hotkeys fields.
const (
	actTimerStart = "timerStart"
	actTimerSync  = "timerSync"
	actTimerReset = "timerReset"
	actOverlay    = "overlay"
)

// profile returns the timer profile with id, or the first one.
func (a *App) profile(id string) gamedata.TimerProfile {
	for _, p := range a.data.Timers {
		if p.ID == id {
			return p
		}
	}
	return a.data.Timers[0] // Validate guarantees at least one
}

func (a *App) startTimer(s config.Settings) {
	c := &a.tmr
	c.mu.Lock()
	c.t = timer.New(a.profile(s.Timer.Profile), nil)
	c.last = c.t.Snapshot()
	c.mu.Unlock()
	a.applyHotkeys(s.Hotkeys)
	go a.runTimer()
}

// runTimer publishes the timer whenever it visibly changes (and once a
// second regardless, so a newly opened overlay catches up) and beeps
// before a phase runs out.
func (a *App) runTimer() {
	tick := time.NewTicker(250 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-tick.C:
		}
		c := &a.tmr
		c.mu.Lock()
		prev, cur := c.last, c.t.Snapshot()
		due := cur != prev || time.Since(c.pub) >= time.Second
		if due {
			c.last, c.pub = cur, time.Now()
		}
		c.mu.Unlock()

		if a.cfg.Get().Timer.Sound {
			if m := timer.Alert(prev, cur, alertMarks); m > 0 {
				go warnBeep(m)
			} else if prev.State == timer.Running && cur.Phase != prev.Phase {
				// A timed phase ran out on its own.
				go phaseBeep()
			}
		}
		if due {
			a.publishTimer(cur)
		}
	}
}

func (a *App) publishTimer(s timer.Snapshot) {
	runtime.EventsEmit(a.ctx, "timer", s)
	a.hub.Publish("timer", s)
}

// timerPhase is the timer's phase for the status event.
func (a *App) timerPhase() string {
	a.tmr.mu.Lock()
	defer a.tmr.mu.Unlock()
	if s := a.tmr.last; s.PhaseID != "" {
		return s.PhaseID
	}
	return string(a.tmr.last.State)
}

// command applies a timer command and publishes the result right away.
func (a *App) command(cmd string) (timer.Snapshot, error) {
	c := &a.tmr
	c.mu.Lock()
	if (cmd == "start" || cmd == "toggle") && c.t.Snapshot().State == timer.Done {
		// Starting over after a finished run is a new expedition.
		defer a.clearExpedition()
	}
	switch cmd {
	case "start":
		c.t.Start()
	case "toggle":
		c.t.Toggle()
	case "pause":
		c.t.Pause()
	case "resume":
		c.t.Resume()
	case "next":
		c.t.Next()
	case "prev":
		c.t.Prev()
	case "reset":
		// Pick up a profile chosen while the run was in progress.
		c.t = timer.New(a.profile(a.cfg.Get().Timer.Profile), nil)
		// A new expedition: forget the bosses of the last one.
		defer a.clearExpedition()
	default:
		c.mu.Unlock()
		return timer.Snapshot{}, fmt.Errorf("unknown timer command %q", cmd)
	}
	s := c.t.Snapshot()
	c.last, c.pub = s, time.Now()
	c.mu.Unlock()
	a.publishTimer(s)
	return s, nil
}

// setTimerProfile switches profile now if no run is in progress;
// otherwise the next reset picks it up.
func (a *App) setTimerProfile(id string) {
	c := &a.tmr
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.t == nil || c.t.Profile().ID == id {
		return
	}
	if s := c.t.Snapshot().State; s == timer.Idle || s == timer.Done {
		c.t = timer.New(a.profile(id), nil)
	}
}

// applyHotkeys (re)registers the global hotkeys if they changed.
func (a *App) applyHotkeys(h config.Hotkeys) {
	c := &a.tmr
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.keys != nil && c.keysCfg == h {
		return
	}
	c.keysCfg = h
	a.listenHotkeysLocked()
}

func (a *App) listenHotkeysLocked() {
	c := &a.tmr
	if c.keys != nil {
		c.keys.Stop()
		c.keys = nil
	}
	c.keyErrs = []string{}
	if !c.suspended {
		h := c.keysCfg
		var errs []error
		c.keys, errs = hotkey.Listen([]hotkey.Binding{
			{Action: actTimerStart, Keys: h.TimerStart},
			{Action: actTimerSync, Keys: h.TimerSync},
			{Action: actTimerReset, Keys: h.TimerReset},
			{Action: actOverlay, Keys: h.Overlay},
		}, func(action string) { go a.onHotkey(action) })
		for _, err := range errs {
			log.Printf("hotkey: %v", err)
			c.keyErrs = append(c.keyErrs, err.Error())
		}
	}
	runtime.EventsEmit(a.ctx, "hotkeys", c.keyErrs)
}

func (a *App) stopHotkeys() {
	c := &a.tmr
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.keys != nil {
		c.keys.Stop()
		c.keys = nil
	}
}

func (a *App) onHotkey(action string) {
	switch action {
	case actTimerStart:
		a.command("toggle")
	case actTimerSync:
		a.command("next")
	case actTimerReset:
		a.command("reset")
	case actOverlay:
		if on, err := a.ToggleOverlay(); err == nil && on {
			runtime.EventsEmit(a.ctx, "overlay:state", true)
		}
	}
}

// timerErr reports why the timer is unavailable: overlay mode, or a
// startup error before the timer was created.
func (a *App) timerErr() error {
	if a.mode != ModeMain {
		return errNotMain
	}
	a.tmr.mu.Lock()
	defer a.tmr.mu.Unlock()
	if a.tmr.t == nil {
		return a.initErr
	}
	return nil
}

// ---- Bound methods ----

func (a *App) GetTimerProfiles() ([]gamedata.TimerProfile, error) {
	if a.mode != ModeMain {
		return nil, errNotMain
	}
	if a.data == nil {
		return nil, a.initErr
	}
	return a.data.Timers, nil
}

func (a *App) TimerState() (timer.Snapshot, error) {
	if err := a.timerErr(); err != nil {
		return timer.Snapshot{}, err
	}
	a.tmr.mu.Lock()
	defer a.tmr.mu.Unlock()
	return a.tmr.t.Snapshot(), nil
}

// TimerCommand runs one of start, toggle, pause, resume, next, prev, reset.
func (a *App) TimerCommand(cmd string) (timer.Snapshot, error) {
	if err := a.timerErr(); err != nil {
		return timer.Snapshot{}, err
	}
	return a.command(cmd)
}

// HotkeyErrors lists hotkeys that could not be registered, e.g. because
// another app already owns them.
func (a *App) HotkeyErrors() []string {
	a.tmr.mu.Lock()
	defer a.tmr.mu.Unlock()
	return append([]string{}, a.tmr.keyErrs...)
}

// SuspendHotkeys turns the global hotkeys off while the settings page
// records a new one; otherwise the old hotkey would swallow the key press.
func (a *App) SuspendHotkeys(on bool) error {
	if err := a.timerErr(); err != nil {
		return err
	}
	c := &a.tmr
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.suspended == on {
		return nil
	}
	c.suspended = on
	a.listenHotkeysLocked()
	return nil
}
