package main

import (
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/Chau9208/nightreign-companion/internal/ipc"
)

// overlayProc manages the overlay child process (same executable, run
// with --overlay).
type overlayProc struct {
	mu  sync.Mutex
	cmd *exec.Cmd
}

func (o *overlayProc) running() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.cmd != nil
}

// start launches the overlay; onExit runs when the process ends for any reason.
func (o *overlayProc) start(hub *ipc.Server, onExit func()) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, "--overlay")
	cmd.Env = append(os.Environ(), hub.Env()...)

	o.mu.Lock()
	defer o.mu.Unlock()
	if o.cmd != nil {
		return nil
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	o.cmd = cmd
	go func() {
		cmd.Wait()
		o.mu.Lock()
		if o.cmd == cmd {
			o.cmd = nil
		}
		o.mu.Unlock()
		onExit()
	}()
	return nil
}

// stop asks the overlay to quit and kills it if it has not exited shortly after.
func (o *overlayProc) stop(hub *ipc.Server) {
	o.mu.Lock()
	cmd := o.cmd
	o.mu.Unlock()
	if cmd == nil {
		return
	}
	if hub != nil {
		hub.Publish("overlay:close", nil)
	}
	time.AfterFunc(2*time.Second, func() {
		o.mu.Lock()
		defer o.mu.Unlock()
		if o.cmd == cmd {
			cmd.Process.Kill()
		}
	})
}
