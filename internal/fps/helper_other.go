//go:build !windows

package fps

import "errors"

// These stubs keep the module building elsewhere; measuring is Windows-only.

var errUnsupported = errors.New("FPS measurement is only supported on Windows")

var ErrCancelled = errUnsupported

type Proc struct{}

func (*Proc) Wait() {}

func LaunchElevated(exe, args string) (*Proc, error) { return nil, errUnsupported }

func FindProcess(exe string) (uint32, error) { return 0, errUnsupported }
