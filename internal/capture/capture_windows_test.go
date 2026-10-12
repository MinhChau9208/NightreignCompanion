package capture

import (
	"image"
	"testing"
)

func TestGrabScreen(t *testing.T) {
	img, err := Grab(image.Rect(0, 0, 64, 32))
	if err != nil {
		t.Fatal(err)
	}
	if img.Rect.Dx() != 64 || img.Rect.Dy() != 32 || img.Pix[3] != 255 {
		t.Fatalf("unexpected capture %v", img.Rect)
	}
}

func TestWindowOfMissingProcess(t *testing.T) {
	// Repeated searches must not create a new callback each time.
	for i := 0; i < 3000; i++ {
		if _, err := Window(0xFFFFFFF0); err != ErrNoWindow {
			t.Fatalf("got %v", err)
		}
	}
}
