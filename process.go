//go:build darwin

package main

import (
	"bufio"
	"bytes"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

type Proc struct {
	PID  int
	PPID int
	User string
	CPU  string
	MEM  string
	Cmd  string
}

func collectProcesses() (map[int]Proc, map[int][]int, error) {
	cmd := exec.Command("/bin/ps", "-axo", "pid=,ppid=,user=,%cpu=,%mem=,command=")
	out, err := cmd.Output()
	if err != nil {
		return nil, nil, err
	}

	processes := make(map[int]Proc)
	children := make(map[int][]int)
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		fields := splitN(scanner.Text(), 6)
		if len(fields) < 5 {
			continue
		}
		if len(fields) == 5 {
			fields = append(fields, "")
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		ppid, err := strconv.Atoi(fields[1])
		if err != nil {
			ppid = 0
		}
		processes[pid] = Proc{PID: pid, PPID: ppid, User: fields[2], CPU: fields[3], MEM: fields[4], Cmd: fields[5]}
		children[ppid] = append(children[ppid], pid)
	}
	for parent := range children {
		sort.Ints(children[parent])
	}
	return processes, children, scanner.Err()
}

func collectCommandNames() (map[int]string, error) {
	cmd := exec.Command("/bin/ps", "-axc", "-o", "pid=", "-o", "comm=")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	names := make(map[int]string)
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		fields := splitN(line, 2)
		if len(fields) == 0 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err == nil {
			names[pid] = strings.TrimSpace(strings.TrimPrefix(line, fields[0]))
		}
	}
	return names, scanner.Err()
}

// splitN splits on whitespace and keeps the remainder in the final field.
func splitN(s string, n int) []string {
	if n <= 0 {
		return nil
	}
	fields := make([]string, 0, n)
	for len(fields) < n-1 {
		s = strings.TrimLeft(s, " 	")
		if s == "" {
			return fields
		}
		end := strings.IndexAny(s, " 	")
		if end < 0 {
			fields = append(fields, s)
			return fields
		}
		fields = append(fields, s[:end])
		s = s[end:]
	}
	if rest := strings.TrimSpace(s); rest != "" {
		fields = append(fields, rest)
	}
	return fields
}

func findRoots(processes map[int]Proc) []int {
	roots := make([]int, 0)
	for pid, process := range processes {
		if _, exists := processes[process.PPID]; !exists {
			roots = append(roots, pid)
		}
	}
	sort.Ints(roots)
	return roots
}

func visiblePIDs(processes map[int]Proc, children map[int][]int, args Args) map[int]struct{} {
	visible := make(map[int]struct{})
	addBranch := func(start int) {
		stack := []int{start}
		for len(stack) > 0 {
			pid := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if _, seen := visible[pid]; !seen {
				visible[pid] = struct{}{}
			}
			stack = append(stack, children[pid]...)
		}
	}
	addAncestors := func(start int) {
		for pid := start; ; {
			process, exists := processes[pid]
			if !exists {
				return
			}
			visible[pid] = struct{}{}
			if process.PPID == 0 {
				return
			}
			pid = process.PPID
		}
	}

	for pid, process := range processes {
		match := args.PID == 0 && args.User == "" && args.Sub == ""
		if args.PID != 0 {
			match = pid == args.PID
		}
		if args.User != "" {
			match = process.User == args.User
		}
		if args.Sub != "" {
			match = strings.Contains(strings.ToLower(process.Cmd), strings.ToLower(args.Sub))
		}
		if match {
			addAncestors(pid)
			addBranch(pid)
		}
	}
	return visible
}
