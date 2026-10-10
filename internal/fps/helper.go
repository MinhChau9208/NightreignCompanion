package fps

import "time"

// SessionName is the ETW session owned by the helper.
const SessionName = "NightreignCompanion-FPS"

// Message types exchanged with the main process over internal/ipc.
const (
	MsgStats = "fps:stats" // helper → main: Status, once per second
	MsgStop  = "fps:stop"  // main → helper: exit
)

// Helper states.
const (
	StateStarting = "starting"
	StateWaiting  = "waiting" // target process not running
	StateRunning  = "running"
	StateError    = "error"
)

// historyLen is how many per-second FPS samples the helper keeps.
const historyLen = 60

// Status is what the helper reports once per second.
type Status struct {
	State   string `json:"state"`
	Process string `json:"process"`
	PID     uint32 `json:"pid"`
	FrameStats
	History []float64 `json:"history"` // FPS per second, oldest first
	Error   string    `json:"error,omitempty"`
}

type HelperConfig struct {
	Addr, Token string // main process IPC endpoint
	Process     string // image name to measure, e.g. nightreign.exe
	Window      time.Duration
}
