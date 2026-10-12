package main

import (
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"time"

	"github.com/MinhChau9208/NightreignCompanion/internal/boss"
	"github.com/MinhChau9208/NightreignCompanion/internal/capture"
	"github.com/MinhChau9208/NightreignCompanion/internal/config"
	"github.com/MinhChau9208/NightreignCompanion/internal/gamedata"
	"github.com/MinhChau9208/NightreignCompanion/internal/ocr"
)

// readBossNames runs boss OCR on a screenshot of the whole game window, so
// screen names and the capture region can be checked without a live fight.
func readBossNames(args []string) {
	fs := flag.NewFlagSet("ocr", flag.ExitOnError)
	def := config.DefaultOCRRegion
	x := fs.Float64("x", def.X, "region left, fraction of the width")
	y := fs.Float64("y", def.Y, "region top, fraction of the height")
	w := fs.Float64("w", def.W, "region width, fraction of the width")
	h := fs.Float64("h", def.H, "region height, fraction of the height")
	full := fs.Bool("full", false, "read the whole image instead of the region")
	fs.Parse(args)
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: nrc-cli ocr [-x -y -w -h | -full] screenshot.png")
		os.Exit(2)
	}

	f, err := os.Open(fs.Arg(0))
	if err != nil {
		fatal(err)
	}
	img, _, err := image.Decode(f)
	f.Close()
	if err != nil {
		fatal(err)
	}
	r := img.Bounds()
	if !*full {
		region := capture.Region{X: *x, Y: *y, W: *w, H: *h}
		if !region.Valid() {
			fatal(fmt.Errorf("region %+v is not inside the image", region))
		}
		r = region.In(r)
	}
	crop := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	for yy := 0; yy < r.Dy(); yy++ {
		for xx := 0; xx < r.Dx(); xx++ {
			crop.Set(xx, yy, img.At(r.Min.X+xx, r.Min.Y+yy))
		}
	}

	eng, err := ocr.New("en-US")
	if err != nil {
		fatal(err)
	}
	defer eng.Close()
	scale := 1
	if r.Dy() < 400 {
		scale = 2
	}
	start := time.Now()
	lines, err := eng.Recognize(ocr.Prepare(crop, scale))
	if err != nil {
		fatal(err)
	}
	fmt.Printf("image %dx%d, region %v (x%d), read in %v\n", img.Bounds().Dx(), img.Bounds().Dy(), r, scale, time.Since(start).Round(time.Millisecond))
	for _, l := range lines {
		fmt.Printf("  | %s\n", l)
	}

	p, err := gamedata.LoadEmbedded()
	if err != nil {
		fatal(err)
	}
	var aliases []boss.Alias
	for _, b := range p.Bosses {
		for _, n := range p.ScreenNames(b.ID) {
			aliases = append(aliases, boss.Alias{ID: b.ID, Name: n})
		}
	}
	if id, score := boss.Match(lines, aliases); id != "" {
		fmt.Printf("match: %s (%.2f)\n", id, score)
	} else {
		fmt.Printf("no boss name (best %.2f, need %.2f)\n", score, boss.MinScore)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
