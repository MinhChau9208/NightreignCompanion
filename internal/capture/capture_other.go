//go:build !windows

package capture

import (
	"errors"
	"image"
)

var errUnsupported = errors.New("screen capture is only available on Windows")

func Window(pid uint32) (image.Rectangle, error) { return image.Rectangle{}, errUnsupported }

func Grab(r image.Rectangle) (*image.RGBA, error) { return nil, errUnsupported }
