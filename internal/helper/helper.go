// Package helper is the body of the elevated helper process (same exe,
// --fps-helper; docs/SCOPE.md §2.4). It owns one ETW session that feeds
// both the FPS tracker (DXGI Present events) and the game-connection
// tracker (Kernel-Network events), and reports both to the main process.
package helper

import "time"

// SessionName is the ETW session owned by the helper.
const SessionName = "NightreignCompanion-FPS"

// MsgStop is the main → helper message asking it to exit.
const MsgStop = "fps:stop"

// historyLen is how many per-second FPS samples the helper keeps.
const historyLen = 60

// maxFlows is how many game connections are reported.
const maxFlows = 8

type Config struct {
	Addr, Token string // main process IPC endpoint
	Process     string // image name to measure, e.g. nightreign.exe
	Window      time.Duration
	NetWindow   time.Duration
}
