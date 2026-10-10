package main

import (
	"cmp"
	"flag"
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/MinhChau9208/NightreignCompanion/internal/etw"
	"github.com/MinhChau9208/NightreignCompanion/internal/gamenet"
	"github.com/MinhChau9208/NightreignCompanion/internal/helper"
)

// measureUDP shows which processes carry UDP traffic, to find out which one
// actually moves a game's co-op data (the game itself, or Steam on its behalf).
func measureUDP(args []string) {
	fs := flag.NewFlagSet("udp", flag.ExitOnError)
	seconds := fs.Int("n", 20, "seconds to watch")
	fs.Parse(args)

	window := time.Duration(*seconds) * time.Second
	local, _ := gamenet.LocalAddrs()
	var (
		mu    sync.Mutex
		procs = map[uint32]*gamenet.Tracker{}
	)
	sess, err := etw.StartSession(helper.SessionName+"-CLI", func(e *etw.Event) {
		p, ok := gamenet.ParseEvent(e)
		if !ok || p.Proto != "udp" {
			return
		}
		mu.Lock()
		tr := procs[p.PID]
		if tr == nil {
			tr = gamenet.NewTracker(window + time.Minute) // keep everything seen
			tr.SetLocal(local)
			procs[p.PID] = tr
		}
		mu.Unlock()
		tr.Add(p)
	})
	if err == nil {
		if err = gamenet.Enable(sess); err != nil {
			sess.Close()
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer sess.Close()
	go sess.Process()

	fmt.Printf("counting UDP per process for %ds…\n", *seconds)
	time.Sleep(window + 1500*time.Millisecond) // + ETW delivery delay

	type row struct {
		pid   uint32
		name  string
		pkts  float64
		flows []gamenet.Flow
	}
	names := processNames()
	now := nowFiletime()
	var rows []row
	mu.Lock()
	for pid, tr := range procs {
		r := row{pid: pid, name: names[pid], flows: tr.Snapshot(now, 5)}
		for _, f := range r.flows {
			r.pkts += f.PktsInPerSec + f.PktsOutPerSec
		}
		rows = append(rows, r)
	}
	mu.Unlock()
	slices.SortFunc(rows, func(a, b row) int { return cmp.Compare(b.pkts, a.pkts) })

	for _, r := range rows[:min(len(rows), 10)] {
		name := r.name
		if name == "" {
			name = "?"
		}
		fmt.Printf("\n%-28s pid %-6d %7.1f pkt/s\n", name, r.pid, r.pkts)
		for _, f := range r.flows {
			fmt.Printf("    %-7s %-28s in %6.1f pkt/s %7.1f kbps   out %6.1f pkt/s %7.1f kbps   gap %5.0fms\n",
				f.Kind, f.Remote, f.PktsInPerSec, f.KbpsIn, f.PktsOutPerSec, f.KbpsOut, f.MaxGapMs)
		}
	}
	if len(rows) == 0 {
		fmt.Println("no UDP traffic seen")
	}
}

// processNames maps PIDs to image names for the processes running now.
func processNames() map[uint32]string {
	out := map[uint32]string{}
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return out
	}
	defer windows.CloseHandle(snap)
	var e windows.ProcessEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	for err = windows.Process32First(snap, &e); err == nil; err = windows.Process32Next(snap, &e) {
		out[e.ProcessID] = strings.ToLower(windows.UTF16ToString(e.ExeFile[:]))
	}
	return out
}
