//go:build darwin

package main

import (
	"fmt"
	"os"
)

func main() {
	args, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr, "Run 'psfax --help' for usage.")
		os.Exit(2)
	}
	if args.Help {
		fmt.Print(usage)
		return
	}

	procs, children, err := collectProcesses()
	if err != nil {
		fmt.Fprintf(os.Stderr, "collect processes: %v\n", err)
		os.Exit(1)
	}
	comm, err := collectCommandNames()
	if err != nil {
		fmt.Fprintf(os.Stderr, "collect command names: %v\n", err)
		os.Exit(1)
	}

	visible := visiblePIDs(procs, children, args)
	printTree(os.Stdout, children, procs, findRoots(procs), visible, args, comm)
}
