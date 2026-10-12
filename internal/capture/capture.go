// Package capture grabs pixels of the game's window from the screen, the
// way recording software does. It finds the window by process id through
// the window manager and never opens or reads the game process.
package capture

import (
	"errors"
	"image"
)

var ErrNoWindow = errors.New("game window not found")

// Region is part of a window in fractions of its client area, so one
// setting works at every resolution.
type Region struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
	H float64 `json:"h"`
}

// In maps r onto a client rectangle in screen pixels.
func (r Region) In(client image.Rectangle) image.Rectangle {
	w, h := float64(client.Dx()), float64(client.Dy())
	return image.Rect(
		client.Min.X+int(r.X*w), client.Min.Y+int(r.Y*h),
		client.Min.X+int((r.X+r.W)*w), client.Min.Y+int((r.Y+r.H)*h),
	).Intersect(client)
}

// Valid reports whether r lies inside the window and is not empty.
func (r Region) Valid() bool {
	return r.X >= 0 && r.Y >= 0 && r.W > 0 && r.H > 0 && r.X+r.W <= 1 && r.Y+r.H <= 1
}
