// Package ocr reads text from screen captures with the OCR engine built
// into Windows (Windows.Media.Ocr), so nothing extra has to be installed.
// It only ever sees pixels the app captured itself; it never touches the
// game process.
package ocr

import (
	"errors"
	"image"
	"image/color"
)

var errUnsupported = errors.New("OCR is only available on Windows")

// Prepare turns a capture into what the OCR engine reads best: grayscale,
// scaled up so small HUD text is at least ~40px tall.
func Prepare(src image.Image, scale int) *image.Gray {
	b := src.Bounds()
	if scale < 1 {
		scale = 1
	}
	w, h := b.Dx(), b.Dy()
	gray := image.NewGray(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			gray.SetGray(x, y, color.GrayModel.Convert(src.At(b.Min.X+x, b.Min.Y+y)).(color.Gray))
		}
	}
	if scale == 1 {
		return gray
	}
	return upscale(gray, scale)
}

// upscale enlarges g by an integer factor with bilinear filtering.
func upscale(g *image.Gray, k int) *image.Gray {
	w, h := g.Rect.Dx(), g.Rect.Dy()
	out := image.NewGray(image.Rect(0, 0, w*k, h*k))
	at := func(x, y int) float64 {
		x = min(max(x, 0), w-1)
		y = min(max(y, 0), h-1)
		return float64(g.Pix[y*g.Stride+x])
	}
	for oy := 0; oy < h*k; oy++ {
		fy := (float64(oy)+0.5)/float64(k) - 0.5
		y0 := int(fy)
		if fy < 0 {
			y0 = -1
		}
		ty := fy - float64(y0)
		for ox := 0; ox < w*k; ox++ {
			fx := (float64(ox)+0.5)/float64(k) - 0.5
			x0 := int(fx)
			if fx < 0 {
				x0 = -1
			}
			tx := fx - float64(x0)
			top := at(x0, y0)*(1-tx) + at(x0+1, y0)*tx
			bot := at(x0, y0+1)*(1-tx) + at(x0+1, y0+1)*tx
			out.Pix[oy*out.Stride+ox] = uint8(top*(1-ty) + bot*ty + 0.5)
		}
	}
	return out
}
