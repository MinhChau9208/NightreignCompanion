package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/MinhChau9208/NightreignCompanion/internal/config"
	"github.com/MinhChau9208/NightreignCompanion/internal/gamedata"
	"github.com/MinhChau9208/NightreignCompanion/internal/ipc"
	"github.com/MinhChau9208/NightreignCompanion/internal/store"
)

type Mode string

const (
	ModeMain    Mode = "main"
	ModeOverlay Mode = "overlay"
)

// App is the single struct bound to the frontend in both processes. Methods
// that only make sense in the main process return errNotMain in the overlay.
type App struct {
	mode    Mode
	ctx     context.Context
	cancel  context.CancelFunc
	started time.Time

	cfgDir string
	cfg    *config.Store
	data   *gamedata.Pack
	db     *store.DB
	hub    *ipc.Server
	ovl    overlayProc
	net    netWatcher

	initErr error
}

var errNotMain = errors.New("only available in the main window")

func NewApp(mode Mode) *App { return &App{mode: mode, started: time.Now()} }

func (a *App) startup(ctx context.Context) {
	a.ctx, a.cancel = context.WithCancel(ctx)
	if err := a.init(); err != nil {
		// Keep the window up so the UI can show what went wrong.
		a.initErr = err
		log.Printf("startup: %v", err)
	}
}

func (a *App) init() error {
	dir, err := config.Dir()
	if err != nil {
		return fmt.Errorf("config dir: %w", err)
	}
	a.cfgDir = dir
	if a.cfg, err = config.Open(filepath.Join(dir, "config.json")); err != nil {
		return err
	}
	if a.mode == ModeOverlay {
		go a.followMain()
		return nil
	}

	if a.data, err = gamedata.LoadEmbedded(); err != nil {
		return fmt.Errorf("game data: %w", err)
	}
	if a.db, err = store.Open(a.ctx, filepath.Join(dir, "nrc.db")); err != nil {
		return fmt.Errorf("database: %w", err)
	}
	if a.hub, err = ipc.Listen(); err != nil {
		return fmt.Errorf("ipc: %w", err)
	}
	go a.publishStatus()
	a.startNet(a.cfg.Get().Network)
	return nil
}

func (a *App) domReady(ctx context.Context) {
	if a.mode == ModeOverlay && a.cfg != nil {
		o := a.cfg.Get().Overlay
		runtime.WindowSetPosition(ctx, o.X, o.Y)
	}
}

func (a *App) shutdown(context.Context) {
	a.cancel()
	if a.mode == ModeMain {
		a.ovl.stop(a.hub)
	}
	if a.hub != nil {
		a.hub.Close()
	}
	if a.db != nil {
		a.db.Close()
	}
}

func (a *App) onSecondInstance(options.SecondInstanceData) {
	runtime.WindowUnminimise(a.ctx)
	runtime.Show(a.ctx)
}

// StatusEvent is broadcast once per second to helper processes.
type StatusEvent struct {
	UptimeSec   int    `json:"uptimeSec"`
	DataVersion string `json:"dataVersion"`
	Phase       string `json:"phase"` // timer phase; "idle" until M2 lands
}

func (a *App) publishStatus() {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-t.C:
			a.hub.Publish("status", StatusEvent{
				UptimeSec:   int(time.Since(a.started).Seconds()),
				DataVersion: a.data.Manifest.DataVersion,
				Phase:       "idle",
			})
		}
	}
}

// followMain streams events from the main process into the overlay's
// frontend, and quits the overlay once the main process is gone.
func (a *App) followMain() {
	addr, token := os.Getenv(ipc.EnvAddr), os.Getenv(ipc.EnvToken)
	const maxFailures = 3
	for failures := 0; failures < maxFailures; failures++ {
		err := ipc.Subscribe(a.ctx, addr, token, func(m ipc.Message) {
			failures = 0
			if m.Type == "overlay:close" {
				runtime.Quit(a.ctx)
				return
			}
			runtime.EventsEmit(a.ctx, m.Type, m.Data)
		})
		if a.ctx.Err() != nil {
			return
		}
		log.Printf("overlay: lost main process: %v", err)
		runtime.EventsEmit(a.ctx, "ipc:disconnected", err.Error())
		time.Sleep(time.Second)
	}
	runtime.Quit(a.ctx)
}

// ---- Bound methods ----

func (a *App) Mode() Mode { return a.mode }

type AppInfo struct {
	Version   string          `json:"version"`
	Mode      Mode            `json:"mode"`
	ConfigDir string          `json:"configDir"`
	Data      *gamedata.Stats `json:"data"`
	Error     string          `json:"error"`
}

func (a *App) Info() AppInfo {
	info := AppInfo{Version: version, Mode: a.mode, ConfigDir: a.cfgDir}
	if a.data != nil {
		s := a.data.Stats()
		info.Data = &s
	}
	if a.initErr != nil {
		info.Error = a.initErr.Error()
	}
	return info
}

func (a *App) GetSettings() (config.Settings, error) {
	if a.cfg == nil {
		return config.Settings{}, a.initErr
	}
	return a.cfg.Get(), nil
}

func (a *App) SaveSettings(s config.Settings) error {
	if a.mode != ModeMain {
		return errNotMain
	}
	if a.cfg == nil {
		return a.initErr
	}
	if err := a.cfg.Save(s); err != nil {
		return err
	}
	a.startNet(s.Network)
	return a.hub.Publish("settings", s)
}

// ToggleOverlay starts or stops the overlay process and reports whether it
// is running afterwards.
func (a *App) ToggleOverlay() (bool, error) {
	if a.mode != ModeMain {
		return false, errNotMain
	}
	if a.hub == nil {
		return false, a.initErr
	}
	if a.ovl.running() {
		a.ovl.stop(a.hub)
		return false, nil
	}
	err := a.ovl.start(a.hub, func() { runtime.EventsEmit(a.ctx, "overlay:state", false) })
	return err == nil, err
}

func (a *App) OverlayRunning() bool { return a.ovl.running() }
