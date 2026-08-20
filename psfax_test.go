//go:build darwin

package main

import (
	"strings"
	"testing"
)

func TestParseArgs(t *testing.T) {
	args, err := parseArgs([]string{"--pid", "42", "--wide"})
	if err != nil || args.PID != 42 || !args.Wide {
		t.Fatalf("parseArgs() = %#v, %v", args, err)
	}
	if _, err := parseArgs([]string{"--pid", "0"}); err == nil {
		t.Fatal("expected invalid PID error")
	}
	if _, err := parseArgs([]string{"--user", "alice", "--sub", "shell"}); err == nil {
		t.Fatal("expected conflicting selector error")
	}
}

func TestSplitNKeepsCommandRemainder(t *testing.T) {
	got := splitN("  12  1 alice  0.1  2.0  /bin/sh -c 'hello world'", 6)
	want := []string{"12", "1", "alice", "0.1", "2.0", "/bin/sh -c 'hello world'"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("splitN() = %#v, want %#v", got, want)
	}
}

func TestVisiblePIDsIncludesAncestorsAndDescendants(t *testing.T) {
	processes := map[int]Proc{
		1: {PID: 1, PPID: 0, User: "root"},
		2: {PID: 2, PPID: 1, User: "alice"},
		3: {PID: 3, PPID: 2, User: "bob"},
		4: {PID: 4, PPID: 1, User: "root"},
	}
	children := map[int][]int{0: {1}, 1: {2, 4}, 2: {3}}
	got := visiblePIDs(processes, children, Args{User: "alice"})
	for _, pid := range []int{1, 2, 3} {
		if _, ok := got[pid]; !ok {
			t.Errorf("PID %d not visible", pid)
		}
	}
	if _, ok := got[4]; ok {
		t.Error("unrelated PID 4 is visible")
	}
}

func TestExecutableOccurrencePreservesUnicodeWindow(t *testing.T) {
	full := "/Applications/Über.app/Contents/MacOS/Über --flag"
	_, occurrence := chooseExecutableOccurrence(full, "Über")
	if occurrence == nil {
		t.Fatal("expected executable occurrence")
	}
	window, start := tailOrAround(full, 12, occurrence)
	if !strings.Contains(window, "Über") {
		t.Fatalf("window %q does not contain executable", window)
	}
	highlighted := highlightInWindow(window, start, occurrence, full, true)
	if !strings.Contains(highlighted, fgExecutable+"Über"+reset) {
		t.Fatalf("highlighted window %q does not highlight executable", highlighted)
	}
}
