package banner

import (
	"image/png"
	"os"
	"testing"

	"github.com/MinhChau9208/NightreignCompanion/internal/ocr"
)

// TestScreenshots reads crops of in-game screenshots (1920x1080) at the
// default banner regions with Windows OCR.
func TestScreenshots(t *testing.T) {
	e, err := ocr.New("en-US")
	if err != nil {
		t.Skipf("no English OCR on this machine: %v", err)
	}
	defer e.Close()
	msgs := []Message{{Text: "The Night's tide grows stronger", Kind: Shrink}}
	for file, want := range map[string]Event{
		"day1.png": {Kind: Day, Day: 1},
		"tide.png": {Kind: Shrink},
	} {
		f, err := os.Open("testdata/" + file)
		if err != nil {
			t.Fatal(err)
		}
		img, err := png.Decode(f)
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		lines, err := e.Recognize(ocr.Prepare(img, 2))
		if err != nil {
			t.Fatal(err)
		}
		ev, ok := Find(lines, msgs)
		if !ok || ev.Kind != want.Kind || ev.Day != want.Day {
			t.Errorf("%s: read %q, got %+v %v, want %s day %d", file, lines, ev, ok, want.Kind, want.Day)
		}
	}
}
