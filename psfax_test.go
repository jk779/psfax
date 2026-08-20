//go:build darwin

package main

import (
	"bytes"
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

func TestPrintTreeUsesContinuationGuides(t *testing.T) {
	processes := map[int]Proc{
		1: {PID: 1, PPID: 0, User: "root", Cmd: "launchd"},
		2: {PID: 2, PPID: 1, User: "alice", Cmd: "first"},
		3: {PID: 3, PPID: 2, User: "alice", Cmd: "grandchild"},
		4: {PID: 4, PPID: 1, User: "alice", Cmd: "second"},
	}
	children := map[int][]int{0: {1}, 1: {2, 4}, 2: {3}}
	visible := map[int]struct{}{1: {}, 2: {}, 3: {}, 4: {}}
	var output bytes.Buffer
	printTree(&output, children, processes, []int{1}, visible, Args{Wide: true}, nil)
	if !strings.Contains(output.String(), "│   └── grandchild") {
		t.Fatalf("tree output lacks continuation guide:\n%s", output.String())
	}
}

func TestChooseExecutableOccurrencePrefersBundleExecutable(t *testing.T) {
	command := "/Applications/Example.app/Contents/MacOS/Example --name Example"
	name, occurrence := chooseExecutableOccurrence(command, "Example")
	if name != "Example" || occurrence == nil || command[occurrence[0]:occurrence[1]] != "Example" {
		t.Fatalf("chooseExecutableOccurrence() = %q, %#v", name, occurrence)
	}
	if got := executableAfterBundle(command); got != "Example" {
		t.Fatalf("executableAfterBundle() = %q", got)
	}
}
