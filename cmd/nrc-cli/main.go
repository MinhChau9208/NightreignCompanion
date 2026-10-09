// Command nrc-cli holds developer utilities for Nightreign Companion.
//
//	nrc-cli validate [dir]                 validate the embedded data pack, or the pack in dir
//	nrc-cli ping [-n count] [target ...]   measure ping/jitter/loss (default: gateway 1.1.1.1 8.8.8.8)
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/MinhChau9208/NightreignCompanion/internal/config"
	"github.com/MinhChau9208/NightreignCompanion/internal/gamedata"
	"github.com/MinhChau9208/NightreignCompanion/internal/netmon"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "validate":
		validate(os.Args[2:])
	case "ping":
		ping(os.Args[2:])
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: nrc-cli validate [dir] | ping [-n count] [target ...]")
	os.Exit(2)
}

func validate(args []string) {
	var (
		p   *gamedata.Pack
		err error
	)
	if len(args) > 0 {
		p, err = gamedata.Load(os.DirFS(args[0]))
	} else {
		p, err = gamedata.LoadEmbedded()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "data pack invalid:\n"+err.Error())
		os.Exit(1)
	}
	b, _ := json.MarshalIndent(p.Stats(), "", "  ")
	fmt.Printf("data pack OK\n%s\n", b)
}

func ping(args []string) {
	fs := flag.NewFlagSet("ping", flag.ExitOnError)
	count := fs.Int("n", 10, "probes per target")
	fs.Parse(args)

	def := config.Defaults()
	targets := fs.Args()
	if len(targets) == 0 {
		targets = def.Network.PingTargets
	}
	th := def.Network.Thresholds
	m := netmon.NewMonitor(netmon.Config{
		Targets:    targets,
		Thresholds: netmon.Thresholds{PingMs: th.PingMs, JitterMs: th.JitterMs, LossPct: th.LossPct},
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*count)*time.Second+500*time.Millisecond)
	defer cancel()
	m.Run(ctx, nil)

	fmt.Printf("%-18s %-16s %8s %8s %8s %8s %7s  %s\n", "TARGET", "ADDRESS", "AVG", "MIN", "MAX", "JITTER", "LOSS", "LEVEL")
	for _, s := range m.Snapshot() {
		if s.Error != "" && s.Sent == 0 {
			fmt.Printf("%-18s error: %s\n", s.Target, s.Error)
			continue
		}
		fmt.Printf("%-18s %-16s %6.1fms %6.1fms %6.1fms %6.1fms %6.1f%%  %s\n",
			s.Target, s.Probe.Address, s.AvgMs, s.MinMs, s.MaxMs, s.JitterMs, s.LossPct, s.Level)
	}
}
