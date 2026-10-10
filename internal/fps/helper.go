package fps

// Helper states, shared by the FPS and game-network reports.
const (
	StateStarting = "starting"
	StateWaiting  = "waiting" // target process not running
	StateRunning  = "running"
	StateError    = "error"
)

// MsgStats is the helper → main message carrying a Status once per second.
const MsgStats = "fps:stats"

// Status is what the helper reports once per second.
type Status struct {
	State   string `json:"state"`
	Process string `json:"process"`
	PID     uint32 `json:"pid"`
	FrameStats
	History []float64 `json:"history"` // FPS per second, oldest first
	Error   string    `json:"error,omitempty"`
}
