//go:build !windows

package fps

import (
	"context"
	"errors"
)

// ETW is Windows-only; these stubs keep the module building elsewhere.

var errUnsupported = errors.New("FPS measurement is only supported on Windows")

var (
	ErrNeedsAdmin = errUnsupported
	ErrCancelled  = errUnsupported
)

type Proc struct{}

func (*Proc) Wait() {}

func LaunchElevated(exe, args string) (*Proc, error) { return nil, errUnsupported }

func FindProcess(exe string) (uint32, error) { return 0, errUnsupported }

func RunHelper(ctx context.Context, cfg HelperConfig) error { return errUnsupported }
