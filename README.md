# Nightreign Companion

Unofficial desktop companion for **ELDEN RING NIGHTREIGN** (Windows): connection & FPS monitor, day/night cycle timer, relic stat lookup, builds, and Nightlord prediction for Deep of Night.

> Fan-made and non-commercial. Not affiliated with FromSoftware or Bandai Namco.
> The app never injects into, reads memory from, or hooks the game, so it stays clear of Easy Anti-Cheat.

Full design: [docs/SCOPE.md](docs/SCOPE.md).

## Status

**Phase 1 — in progress.** Foundation (Phase 0) is done; the network monitor (ping, jitter, packet loss) and FPS measurement work. Timer and relic lookup are next.

## Requirements

- Go 1.26+
- Node.js 22+
- Wails CLI v2: `go install github.com/wailsapp/wails/v2/cmd/wails@v2.10.1`
- WebView2 runtime (included in Windows 11)

## Development

```sh
wails dev                       # run with hot reload
wails build                     # build/bin/nightreign-companion.exe
go test ./...                   # backend tests
go run ./cmd/nrc-cli validate   # validate the bundled data pack
go run ./cmd/nrc-cli ping       # measure ping/jitter/loss from the terminal
go run ./cmd/nrc-cli fps        # measure the game's FPS (Administrator terminal)
cd frontend && npm run check    # frontend type-check
```

## Layout

```
main.go, app.go, overlay.go   Wails entry; App is bound to the frontend in both processes
internal/config               user settings (%AppData%\NightreignCompanion\config.json)
internal/gamedata             data pack loader + validator
internal/store                SQLite (pure Go) — runs, builds
internal/ipc                  event bus between main and overlay processes
internal/netmon               ping (ICMP/TCP), jitter, packet loss, gateway detection
internal/fps                  FPS from DXGI Present events via ETW; elevated helper
data/                         bundled data pack (JSON)
cmd/nrc-cli                   developer utilities
frontend/                     Svelte + TypeScript UI
```

FPS is measured by the same executable started elevated (`--fps-helper`) through a UAC prompt; only that helper runs as Administrator.

The overlay is a second process (`nightreign-companion.exe --overlay`) because Wails v2 supports one window per process. The main process owns all state and streams events to it over a token-protected loopback connection.

## Data accuracy

Every entry in `data/` has a `verified` flag. Seed values are placeholders until they are checked against a datamine or measured in game; the UI marks unverified data.
