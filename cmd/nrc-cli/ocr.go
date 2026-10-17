package main

import (
	"flag"
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"
	"time"

	"github.com/MinhChau9208/NightreignCompanion/internal/banner"
	"github.com/MinhChau9208/NightreignCompanion/internal/boss"
	"github.com/MinhChau9208/NightreignCompanion/internal/capture"
	"github.com/MinhChau9208/NightreignCompanion/internal/config"
	"github.com/MinhChau9208/NightreignCompanion/internal/gamedata"
	"github.com/MinhChau9208/NightreignCompanion/internal/ocr"
)

// readBossNames runs boss OCR on a screenshot of the whole game window, so
// screen names and the capture region can be checked without a live fight.
// With -banner it looks for the timer's Day banner and top-right messages
// instead, reading both default banner regions unless one is given.
func readBossNames(args []string) {
	fs := flag.NewFlagSet("ocr", flag.ExitOnError)
	isBanner := fs.Bool("banner", false, "look for the Day banner and Night's Tide messages")
	x := fs.Float64("x", 0, "region left, fraction of the width")
	y := fs.Float64("y", 0, "region top, fraction of the height")
	w := fs.Float64("w", 0, "region width, fraction of the width")
	h := fs.Float64("h", 0, "region height, fraction of the height")
	full := fs.Bool("full", false, "read the whole image instead of the region")
	fs.Parse(args)
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: nrc-cli ocr [-banner] [-x -y -w -h | -full] screenshot.png")
		os.Exit(2)
	}
	custom := false
	fs.Visit(func(f *flag.Flag) { custom = custom || strings.Contains("xywh", f.Name) && len(f.Name) == 1 })

	f, err := os.Open(fs.Arg(0))
	if err != nil {
		fatal(err)
	}
	img, _, err := image.Decode(f)
	f.Close()
	if err != nil {
		fatal(err)
	}
	var regions []capture.Region
	switch {
	case *full:
		regions = []capture.Region{{X: 0, Y: 0, W: 1, H: 1}}
	case custom:
		regions = []capture.Region{{X: *x, Y: *y, W: *w, H: *h}}
	case *isBanner:
		regions = []capture.Region{config.DefaultDayRegion, config.DefaultMessageRegion}
	default:
		regions = []capture.Region{config.DefaultOCRRegion}
	}

	eng, err := ocr.New("en-US")
	if err != nil {
		fatal(err)
	}
	defer eng.Close()
	var lines []string
	for _, region := range regions {
		if !region.Valid() {
			fatal(fmt.Errorf("region %+v is not inside the image", region))
		}
		r := region.In(img.Bounds())
		crop := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
		draw.Draw(crop, crop.Bounds(), img, r.Min, draw.Src)
		scale := 1
		if r.Dy() < 400 {
			scale = 2
		}
		start := time.Now()
		l, err := eng.Recognize(ocr.Prepare(crop, scale))
		if err != nil {
			fatal(err)
		}
		fmt.Printf("image %dx%d, region %v (x%d), read in %v\n", img.Bounds().Dx(), img.Bounds().Dy(), r, scale, time.Since(start).Round(time.Millisecond))
		for _, line := range l {
			fmt.Printf("  | %s\n", line)
		}
		lines = append(lines, l...)
	}

	p, err := gamedata.LoadEmbedded()
	if err != nil {
		fatal(err)
	}
	if *isBanner {
		var msgs []banner.Message
		for _, m := range p.Messages.Messages {
			msgs = append(msgs, banner.Message{Text: m.Text, Kind: m.Event})
		}
		if ev, ok := banner.Find(lines, msgs); ok {
			fmt.Printf("banner: %s day=%d %q\n", ev.Kind, ev.Day, ev.Text)
		} else {
			fmt.Println("no banner")
		}
		return
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
