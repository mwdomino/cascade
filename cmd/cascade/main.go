package main

import (
	"flag"
	"fmt"
	"os"
)

var version = "0.0.1-dev"

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Printf("cascade %s\n", version)
		return
	}
	// TUI entrypoint added in Task 11.
	fmt.Fprintln(os.Stderr, "cascade: TUI not yet wired (Task 11)")
	os.Exit(1)
}
