package main

import (
	"embed"
	"flag"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

// version is overridden at build time: -ldflags "-X main.version=1.2.3".
var version = "0.1.2-dev"

func main() {
	overlay := flag.Bool("overlay", false, "run as the in-game overlay window (started by the main app)")
	fpsHelper := flag.Bool("fps-helper", false, "run as the elevated FPS helper (started by the main app)")
	ipcAddr := flag.String("ipc-addr", "", "main process IPC address (fps helper)")
	ipcToken := flag.String("ipc-token", "", "main process IPC token (fps helper)")
	process := flag.String("process", "nightreign.exe", "process to measure (fps helper)")
	flag.Parse()

	var err error
	switch {
	case *fpsHelper:
		err = runFPSHelper(*ipcAddr, *ipcToken, *process)
	case *overlay:
		err = runOverlay()
	default:
		err = runMain()
	}
	if err != nil {
		log.Fatal(err)
	}
}

func runMain() error {
	app := NewApp(ModeMain)
	return wails.Run(&options.App{
		Title:            "Nightreign Companion",
		Width:            1100,
		Height:           720,
		MinWidth:         900,
		MinHeight:        600,
		AssetServer:      &assetserver.Options{Assets: assets},
		BackgroundColour: &options.RGBA{R: 14, G: 16, B: 22, A: 255},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId:               "nightreign-companion-main",
			OnSecondInstanceLaunch: app.onSecondInstance,
		},
		Bind: []any{app},
	})
}

func runOverlay() error {
	app := NewApp(ModeOverlay)
	return wails.Run(&options.App{
		Title:            "Nightreign Companion Overlay",
		Width:            300,
		Height:           170,
		DisableResize:    true,
		Frameless:        true,
		AlwaysOnTop:      true,
		AssetServer:      &assetserver.Options{Assets: assets},
		BackgroundColour: &options.RGBA{R: 0, G: 0, B: 0, A: 0},
		OnStartup:        app.startup,
		OnDomReady:       app.domReady,
		OnShutdown:       app.shutdown,
		// The overlay's opacity is the alpha of its CSS background, so both the
		// webview and the host window must be see-through. WindowIsTranslucent
		// with BackdropType None gives a plain transparent window (no Mica or
		// Acrylic blur); without it the host window paints solid black.
		Windows: &windows.Options{
			WebviewIsTransparent:              true,
			WindowIsTranslucent:               true,
			BackdropType:                      windows.None,
			DisableFramelessWindowDecorations: true,
		},
		Bind: []any{app},
	})
}
