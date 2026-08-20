//go:build darwin

package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"unicode/utf8"
	"unsafe"
)

const (
	userColumnWidth                            = 10
	ellipsis                                   = "…"
	fgPID, fgUser, fgTree, fgExecutable, reset = "\x1b[97m", "\x1b[92m", "\x1b[90m", "\x1b[1;33m", "\x1b[0m"
)

func printTree(w io.Writer, children map[int][]int, processes map[int]Proc, roots []int, visible map[int]struct{}, args Args, commandNames map[int]string) {
	termWidth := terminalWidth()
	fmt.Fprintf(w, "%5s %5s %5s %*s  %s\n", "PID", "%CPU", "%MEM", userColumnWidth, "USER", "COMMAND")
	useColor := isTTY()

	var walk func(int, string, string)
	walk = func(pid int, ancestors, connector string) {
		if _, ok := visible[pid]; !ok {
			return
		}
		process := processes[pid]
		pidColumn := fmt.Sprintf("%5d", process.PID)
		userColumn := fmt.Sprintf("%*s", userColumnWidth, truncateUser(process.User, userColumnWidth))
		left := fmt.Sprintf("%s %5s %5s %s  ", pidColumn, process.CPU, process.MEM, userColumn)
		maxCommand := termWidth - displayWidth(left) - displayWidth(ancestors) - displayWidth(connector) - 1
		if maxCommand < 10 {
			maxCommand = 10
		}

		_, occurrence := chooseExecutableOccurrence(process.Cmd, commandNames[pid])
		var command string
		var start int
		if args.Wide {
			command = process.Cmd
		} else {
			command, start = tailOrAround(process.Cmd, maxCommand, occurrence)
		}
		command = highlightInWindow(command, start, occurrence, process.Cmd, useColor)

		leftCol := colorize(pidColumn, fgPID, useColor) + " " + fmt.Sprintf("%5s %5s ", process.CPU, process.MEM) + colorize(userColumn, fgUser, useColor) + "  "
		treeCol := colorize(ancestors+connector, fgTree, useColor)
		fmt.Fprintln(w, leftCol+treeCol+command)

		kids := visibleChildren(children[pid], visible)
		for i, child := range kids {
			childConnector := "├── "
			if i == len(kids)-1 {
				childConnector = "└── "
			}
			childAncestors := ancestors
			if connector == "├── " {
				childAncestors += "│   "
			}
			if connector == "└── " {
				childAncestors += "    "
			}
			walk(child, childAncestors, childConnector)
		}
	}
	for _, root := range roots {
		walk(root, "", "")
	}
}

func visibleChildren(children []int, visible map[int]struct{}) []int {
	out := make([]int, 0, len(children))
	for _, child := range children {
		if _, ok := visible[child]; ok {
			out = append(out, child)
		}
	}
	return out
}

func isTTY() bool {
	info, err := os.Stdout.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func colorize(value, code string, enabled bool) string {
	if !enabled {
		return value
	}
	return code + value + reset
}

func displayWidth(value string) int { return utf8.RuneCountInString(value) }

func truncateUser(value string, width int) string {
	runes := []rune(value)
	if len(runes) <= width {
		return value
	}
	if width <= 1 {
		return string(runes[:width])
	}
	return string(runes[:width-1]) + ellipsis
}

func chooseExecutableOccurrence(full, commandName string) (string, *[2]int) {
	for _, candidate := range []string{commandName, executableAfterBundle(full), argv0Base(full)} {
		if candidate == "" {
			continue
		}
		if start, end, err := findBestOccurrence(full, candidate); err == nil {
			return candidate, &[2]int{start, end}
		}
	}
	return "", nil
}

func findBestOccurrence(full, commandName string) (int, int, error) {
	if full == "" || commandName == "" {
		return 0, 0, errors.New("no occurrence")
	}
	patterns := []struct {
		expression string
		group      int
	}{
		{`/Contents/MacOS/(` + regexp.QuoteMeta(commandName) + `)($|[^A-Za-z0-9._-])`, 1},
		{`(^|[\s/])(` + regexp.QuoteMeta(commandName) + `)($|[^A-Za-z0-9._-])`, 2},
		{`(` + regexp.QuoteMeta(commandName) + `)($|[^A-Za-z0-9._-])`, 1},
	}
	for _, pattern := range patterns {
		match := regexp.MustCompile(pattern.expression).FindStringSubmatchIndex(full)
		if match == nil {
			continue
		}
		index := pattern.group * 2
		if index+1 < len(match) && match[index] >= 0 {
			return match[index], match[index+1], nil
		}
	}
	return 0, 0, errors.New("no occurrence")
}

func executableAfterBundle(command string) string {
	return textAfterMarker(command, "/Contents/MacOS/")
}

func textAfterMarker(value, marker string) string {
	index := strings.LastIndex(value, marker)
	if index < 0 {
		return ""
	}
	result := value[index+len(marker):]
	if cut := strings.Index(result, " --"); cut >= 0 {
		result = result[:cut]
	}
	if cut := strings.Index(result, " /"); cut >= 0 {
		result = result[:cut]
	}
	return strings.TrimSpace(result)
}

func argv0Base(command string) string {
	command = strings.TrimLeft(command, " \t")
	if command == "" {
		return ""
	}
	if cut := strings.IndexAny(command, " \t"); cut >= 0 {
		command = command[:cut]
	}
	command = strings.TrimLeft(command, "-")
	if slash := strings.LastIndex(command, "/"); slash >= 0 {
		command = command[slash+1:]
	}
	return command
}

// tailOrAround keeps byte offsets for occurrence matching but slices by runes.
func tailOrAround(full string, width int, occurrence *[2]int) (string, int) {
	runes := []rune(full)
	if width <= 0 || width >= len(runes) {
		return full, 0
	}
	tailStart := len(runes) - width
	if occurrence == nil {
		return runeWindow(runes, tailStart, width), tailStart
	}
	occurrenceStart := utf8.RuneCountInString(full[:occurrence[0]])
	if occurrenceStart >= tailStart {
		return runeWindow(runes, tailStart, width), tailStart
	}
	start := occurrenceStart - 3
	if start < 0 {
		start = 0
	}
	return runeWindow(runes, start, width), start
}

func runeWindow(full []rune, start, width int) string {
	end := start + width
	if end > len(full) {
		end = len(full)
		start = end - width
	}
	window := append([]rune(nil), full[start:end]...)
	if start > 0 {
		window[0] = []rune(ellipsis)[0]
	}
	if end < len(full) {
		window[len(window)-1] = []rune(ellipsis)[0]
	}
	return string(window)
}

func highlightInWindow(window string, start int, occurrence *[2]int, full string, enabled bool) string {
	if !enabled || occurrence == nil {
		return window
	}
	startRune := utf8.RuneCountInString(full[:occurrence[0]])
	endRune := utf8.RuneCountInString(full[:occurrence[1]])
	from, to := startRune-start, endRune-start
	runes := []rune(window)
	if from < 0 || to > len(runes) || from >= to {
		return window
	}
	return string(runes[:from]) + colorize(string(runes[from:to]), fgExecutable, true) + string(runes[to:])
}

func terminalWidth() int {
	if width, ok := ttyWidth(); ok && width >= 40 {
		return width
	}
	if columns, err := strconv.Atoi(os.Getenv("COLUMNS")); err == nil && columns >= 40 {
		return columns
	}
	return 120
}

type windowSize struct{ Row, Col, Xpixel, Ypixel uint16 }

func ttyWidth() (int, bool) {
	const tiocgwinsz = 0x40087468
	for _, fd := range []uintptr{uintptr(syscall.Stdout), uintptr(syscall.Stderr)} {
		ws := windowSize{}
		_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, tiocgwinsz, uintptr(unsafe.Pointer(&ws)))
		if errno == 0 && ws.Col > 0 {
			return int(ws.Col), true
		}
	}
	return 0, false
}
