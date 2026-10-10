// Command nrc-cli holds developer utilities for Nightreign Companion.
//
//	nrc-cli validate [dir]                 validate the embedded data pack, or the pack in dir
//	nrc-cli ping [-n count] [target ...]   measure ping/jitter/loss (default: gateway 1.1.1.1 8.8.8.8)
//	nrc-cli fps [-n seconds] [process]     measure FPS via ETW (run from an Administrator terminal)
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/MinhChau9208/NightreignCompanion/internal/config"
	"github.com/MinhChau9208/NightreignCompanion/internal/fps"
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
	case "fps":
		measureFPS(os.Args[2:])
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: nrc-cli validate [dir] | ping [-n count] [target ...] | fps [-n seconds] [process]")
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

func measureFPS(args []string) {
	fs := flag.NewFlagSet("fps", flag.ExitOnError)
	seconds := fs.Int("n", 15, "seconds to measure")
	fs.Parse(args)
	process := config.Defaults().FPS.Process
	if fs.NArg() > 0 {
		process = fs.Arg(0)
	}

	pid, err := fps.FindProcess(process)
	if err != nil || pid == 0 {
		fmt.Fprintf(os.Stderr, "%s is not running (%v)\n", process, err)
		os.Exit(1)
	}
	tr := fps.NewTracker(30 * time.Second)
	sess, err := fps.StartSession(fps.SessionName+"-CLI", func(p uint32, ts int64) {
		if p == pid {
			tr.Add(ts)
		}
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer sess.Close()
	go sess.Process()

	fmt.Printf("measuring %s (pid %d) for %ds\n", process, pid, *seconds)
	fmt.Printf("%6s %8s %8s %8s %10s %10s\n", "SEC", "FPS", "AVG", "1%LOW", "FRAME ms", "MAX ms")
	for i := 1; i <= *seconds; i++ {
		time.Sleep(time.Second)
		ft := time.Now().UnixNano()/100 + 116444736000000000 // Unix → FILETIME
		s := tr.Stats(ft)
		fmt.Printf("%6d %8.0f %8.1f %8.1f %10.2f %10.2f\n", i, s.FPS, s.AvgFPS, s.Low1FPS, s.FrametimeMs, s.MaxFrametimeMs)
	}
}
