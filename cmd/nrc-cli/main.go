// Command nrc-cli holds developer utilities for Nightreign Companion.
//
//	nrc-cli validate [dir]   validate the embedded data pack, or the pack in dir
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/MinhChau9208/NightreignCompanion/internal/gamedata"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "validate":
		validate(os.Args[2:])
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: nrc-cli validate [dir]")
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
