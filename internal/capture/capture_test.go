package capture

import (
	"image"
	"testing"
)

func TestRegionIn(t *testing.T) {
	client := image.Rect(100, 50, 2020, 1130) // 1920x1080 at (100,50)
	r := Region{X: 0.25, Y: 0.5, W: 0.5, H: 0.25}.In(client)
	if want := image.Rect(580, 590, 1540, 860); r != want {
		t.Errorf("got %v, want %v", r, want)
	}
	if !(Region{X: 0.2, Y: 0.7, W: 0.4, H: 0.2}).Valid() || (Region{X: 0.8, Y: 0, W: 0.4, H: 1}).Valid() {
		t.Error("Valid")
	}
}
