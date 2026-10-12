package ocr

import (
	"image/png"
	"os"
	"strings"
	"testing"
)

func TestRecognizeBossBar(t *testing.T) {
	f, err := os.Open("testdata/bossbar.png")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	e, err := New("en-US")
	if err != nil {
		t.Skipf("no English OCR on this machine: %v", err)
	}
	defer e.Close()
	lines, err := e.Recognize(Prepare(img, 2))
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Join(lines, "\n")
	for _, want := range []string{"Queen Marigga", "Swordmaster Onze"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in:\n%s", want, text)
		}
	}
}
