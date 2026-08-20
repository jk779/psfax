//go:build darwin

package main

import (
	"errors"
	"fmt"
	"strconv"
)

const usage = `psfax displays a macOS process tree, similar to Linux ps fax.

Usage:
  psfax [options]

Options:
  -p, --pid PID       Show the selected process and its descendants.
  -u, --user USER     Show branches containing processes owned by USER.
  -s, --sub TEXT      Show branches whose command contains TEXT.
      --wide          Do not truncate command lines.
      --version       Show the psfax version.
  -h, --help          Show this help.
`

type Args struct {
	PID     int
	User    string
	Sub     string
	Wide    bool
	Version bool
	Help    bool
}

func parseArgs(args []string) (Args, error) {
	var out Args
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-p", "--pid":
			value, next, err := nextValue(args, i, "pid")
			if err != nil {
				return Args{}, err
			}
			pid, err := strconv.Atoi(value)
			if err != nil || pid <= 0 {
				return Args{}, fmt.Errorf("invalid pid %q", value)
			}
			out.PID, i = pid, next
		case "-u", "--user":
			value, next, err := nextValue(args, i, "user")
			if err != nil {
				return Args{}, err
			}
			out.User, i = value, next
		case "-s", "--sub":
			value, next, err := nextValue(args, i, "substring")
			if err != nil {
				return Args{}, err
			}
			out.Sub, i = value, next
		case "--wide":
			out.Wide = true
		case "--version":
			out.Version = true
		case "-h", "--help":
			out.Help = true
		default:
			return Args{}, fmt.Errorf("unknown option %q", args[i])
		}
	}

	selectors := 0
	if out.PID != 0 {
		selectors++
	}
	if out.User != "" {
		selectors++
	}
	if out.Sub != "" {
		selectors++
	}
	if selectors > 1 {
		return Args{}, errors.New("only one of --pid, --user, and --sub may be used")
	}
	return out, nil
}

func nextValue(args []string, index int, name string) (string, int, error) {
	if index+1 >= len(args) || args[index+1] == "" {
		return "", index, fmt.Errorf("missing value for --%s", name)
	}
	return args[index+1], index + 1, nil
}
