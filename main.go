package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"unicode/utf8"
	"unsafe"
)

const (
	userColWidth = 10
	ellipsis     = "…"

	fgPID  = "\x1b[97m"
	fgUser = "\x1b[92m"
	fgTree = "\x1b[90m"
	fgExe  = "\x1b[1;33m"
	reset  = "\x1b[0m"
)

type Proc struct {
	PID  int
	PPID int
	User string
	CPU  string
	MEM  string
	Cmd  string
}

type Args struct {
	Pid  int
	User string
	Sub  string
	Wide bool
}

func main() {
	args := parseArgs()
	procs, children, err := collectProcs()
	if err != nil {
		fmt.Fprintln(os.Stderr, "ps failed:", err)
		os.Exit(1)
	}
	pid2comm, err := collectComm()
	if err != nil {
		fmt.Fprintln(os.Stderr, "ps -axc failed:", err)
		os.Exit(1)
	}

	roots := findRoots(procs)
	visible := buildVisibleSet(children, procs, args)

	printTree(children, procs, roots, visible, args, pid2comm)
}

func parseArgs() Args {
	var out Args
	for i := 1; i < len(os.Args); i++ {
		a := os.Args[i]
		switch a {
		case "-p", "--pid":
			if i+1 >= len(os.Args) {
				die("missing value for -p")
			}
			p, err := strconv.Atoi(os.Args[i+1])
			if err != nil {
				die("invalid pid")
			}
			out.Pid = p
			i++
		case "-u", "--user":
			if i+1 >= len(os.Args) {
				die("missing value for -u")
			}
			out.User = os.Args[i+1]
			i++
		case "-s", "--sub":
			if i+1 >= len(os.Args) {
				die("missing value for -s")
			}
			out.Sub = os.Args[i+1]
			i++
		case "--wide":
			out.Wide = true
		default:
			if strings.HasPrefix(a, "-") {
				die("unknown flag: " + a)
			}
		}
	}
	return out
}

func die(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(2)
}

func collectProcs() (map[int]Proc, map[int][]int, error) {
	cmd := exec.Command("/bin/ps", "-axo", "pid=,ppid=,user=,%cpu=,%mem=,command=")
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	out, err := cmd.Output()
	if err != nil {
		return nil, nil, err
	}
	meta := make(map[int]Proc)
	children := make(map[int][]int)

	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		// split into 6 fields (pid,ppid,user,%cpu,%mem,command)
		parts := splitN(line, 6)
		if len(parts) < 5 {
			continue
		}
		if len(parts) == 5 {
			parts = append(parts, "")
		}
		pid, err1 := strconv.Atoi(parts[0])
		ppid, err2 := strconv.Atoi(parts[1])
		if err1 != nil {
			continue
		}
		if err2 != nil {
			ppid = 0
		}
		pr := Proc{
			PID:  pid,
			PPID: ppid,
			User: parts[2],
			CPU:  parts[3],
			MEM:  parts[4],
			Cmd:  parts[5],
		}
		meta[pid] = pr
		children[ppid] = append(children[ppid], pid)
	}
	for k := range children {
		sort.Ints(children[k])
	}
	return meta, children, sc.Err()
}

func collectComm() (map[int]string, error) {
	cmd := exec.Command("/bin/ps", "-axc", "-o", "pid=", "-o", "comm=")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	m := make(map[int]string)
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		parts := splitN(line, 2)
		if len(parts) == 0 {
			continue
		}
		pid, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		comm := ""
		if len(parts) > 1 {
			comm = parts[1]
		}
		m[pid] = comm
	}
	return m, sc.Err()
}

func splitN(s string, n int) []string {
	// Like Python's split(None, n-1); keep the remainder in the last field.
	fields := []string{}
	cur := ""
	count := 0
	inSpace := true
	for _, r := range s {
		if r == ' ' || r == '\t' {
			if !inSpace {
				if count < n-1 {
					fields = append(fields, cur)
					cur = ""
					count++
				} else {
					cur += string(r)
				}
			} else if count >= n-1 {
				cur += string(r)
			}
			inSpace = true
		} else {
			cur += string(r)
			inSpace = false
		}
	}
	if cur != "" || count < n {
		fields = append(fields, strings.TrimSpace(cur))
	}
	return fields
}

func findRoots(meta map[int]Proc) []int {
	pids := make(map[int]struct{}, len(meta))
	for pid := range meta {
		pids[pid] = struct{}{}
	}
	var roots []int
	for pid, pr := range meta {
		if _, ok := pids[pr.PPID]; !ok {
			roots = append(roots, pid)
		}
	}
	sort.Ints(roots)
	return roots
}

func buildVisibleSet(children map[int][]int, meta map[int]Proc, a Args) map[int]struct{} {
	visible := make(map[int]struct{})

	addBranch := func(start int) {
		stack := []int{start}
		for len(stack) > 0 {
			cur := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if _, ok := visible[cur]; ok {
				continue
			}
			visible[cur] = struct{}{}
			for _, c := range children[cur] {
				stack = append(stack, c)
			}
		}
	}
	addAnc := func(start int) {
		cur := start
		for {
			pr, ok := meta[cur]
			if !ok {
				break
			}
			if _, seen := visible[cur]; seen {
				break
			}
			visible[cur] = struct{}{}
			if pr.PPID == 0 {
				break
			}
			cur = pr.PPID
		}
	}

	if a.Pid != 0 {
		if _, ok := meta[a.Pid]; ok {
			addAnc(a.Pid)
			addBranch(a.Pid)
		}
		return visible
	}
	if a.User != "" {
		for pid, pr := range meta {
			if pr.User == a.User {
				addAnc(pid)
				addBranch(pid)
			}
		}
		return visible
	}
	if a.Sub != "" {
		sub := strings.ToLower(a.Sub)
		for pid, pr := range meta {
			if strings.Contains(strings.ToLower(pr.Cmd), sub) {
				addAnc(pid)
				addBranch(pid)
			}
		}
		return visible
	}
	for pid := range meta {
		visible[pid] = struct{}{}
	}
	return visible
}

func isTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func colorize(s, code string, enabled bool) string {
	if !enabled {
		return s
	}
	return code + s + reset
}

func padLeft(s string, width int) string {
	w := displayWidth(s)
	if w >= width {
		return s
	}
	return strings.Repeat(" ", width-w) + s
}

func displayWidth(s string) int {
	return utf8.RuneCountInString(s)
}

func truncUser(u string, width int) string {
	if displayWidth(u) <= width {
		return u
	}
	if width <= 1 {
		return u[:width]
	}
	runes := []rune(u)
	return string(runes[:width-1]) + ellipsis
}

// ---------- Occurrence selection ----------
func findBestOccurrence(full, comm string) (int, int, error) {
	if full == "" || comm == "" {
		return 0, 0, errors.New("no occurrence")
	}

	// Helper: return indices of the specified capture group (1-based).
	grabGroup := func(re *regexp.Regexp, s string, execGroup int) (int, int, bool) {
		loc := re.FindStringSubmatchIndex(s)
		if loc == nil {
			return 0, 0, false
		}
		// loc layout: [fullS,fullE, g1S,g1E, g2S,g2E, ...]
		idx := 2 * execGroup
		if len(loc) <= idx+1 {
			return 0, 0, false
		}
		start, end := loc[idx], loc[idx+1]
		if start < 0 || end < 0 || start >= end {
			return 0, 0, false
		}
		return start, end, true
	}

	// 1) immediately after /Contents/MacOS/
	//    Pattern: /Contents/MacOS/(exec)(right)
	pat1 := regexp.MustCompile(`/Contents/MacOS/(` + regexp.QuoteMeta(comm) + `)($|[^A-Za-z0-9._-])`)
	if i, j, ok := grabGroup(pat1, full, 1); ok { // exec = group 1
		return i, j, nil
	}

	// 2) component boundary on the left (start/space/slash) + token boundary on the right
	//    Pattern: (left)(exec)(right)
	pat2 := regexp.MustCompile(`(^|[\s/])(` + regexp.QuoteMeta(comm) + `)($|[^A-Za-z0-9._-])`)
	if i, j, ok := grabGroup(pat2, full, 2); ok { // exec = group 2
		return i, j, nil
	}

	// 3) fallback: anywhere, but token boundary on the right
	//    Pattern: (exec)(right)
	pat3 := regexp.MustCompile(`(` + regexp.QuoteMeta(comm) + `)($|[^A-Za-z0-9._-])`)
	if i, j, ok := grabGroup(pat3, full, 1); ok { // exec = group 1
		return i, j, nil
	}

	return 0, 0, errors.New("no occurrence")
}

// ---- target selection & fallbacks (comm → MacOS exe → argv0 base) ----
func lastAfter(full, marker string) (string, bool) {
	idx := strings.LastIndex(full, marker)
	if idx < 0 {
		return "", false
	}
	cut := full[idx+len(marker):]
	// stop before args if obvious
	if p := strings.Index(cut, " --"); p >= 0 {
		cut = cut[:p]
	}
	if p := strings.Index(cut, " /"); p >= 0 {
		cut = cut[:p]
	}
	return strings.TrimSpace(cut), cut != ""
}

func baseOfArgv0(full string) string {
	fs := strings.TrimLeft(full, " \t")
	if fs == "" {
		return ""
	}
	tok := fs
	if sp := strings.IndexAny(fs, " \t"); sp >= 0 {
		tok = fs[:sp]
	}
	tok = strings.TrimLeft(tok, "-")
	if slash := strings.LastIndex(tok, "/"); slash >= 0 {
		tok = tok[slash+1:]
	}
	return tok
}

func chooseTargetAndOcc(full, comm string) (string, *[2]int) {
	if comm != "" {
		if i, j, err := findBestOccurrence(full, comm); err == nil {
			return comm, &[2]int{i, j}
		}
	}
	if bin, ok := lastAfter(full, "/Contents/MacOS/"); ok && bin != "" {
		if i, j, err := findBestOccurrence(full, bin); err == nil {
			return bin, &[2]int{i, j}
		}
	}
	if base := baseOfArgv0(full); base != "" {
		if i, j, err := findBestOccurrence(full, base); err == nil {
			return base, &[2]int{i, j}
		}
	}
	return "", nil
}

// ---------- Unified window building ----------
func buildWindow(full string, start, width int) (string, int) {
	n := len(full)
	if width >= n {
		return full, 0
	}
	if start < 0 {
		start = 0
	}
	end := start + width
	if end > n {
		end = n
		start = n - width
	}
	slice := []rune(full[start:end])
	// Replace first/last rune with ellipsis when trimmed
	if start > 0 && len(slice) > 0 {
		slice[0] = []rune(ellipsis)[0]
	}
	if end < n && len(slice) > 0 {
		slice[len(slice)-1] = []rune(ellipsis)[0]
	}
	return string(slice), start
}

func tailOrAround(full string, width int, occStart, occEnd int, haveOcc bool) (string, int) {
	n := len(full)
	if width >= n || width <= 0 {
		return full, 0
	}
	// Tail window
	tailStart := n - width
	tailVis, tailStart := buildWindow(full, tailStart, width)

	if haveOcc {
		if occStart >= tailStart {
			return tailVis, tailStart
		}
		// small pre-context
		pre := 3
		start := occStart - pre
		if start < 0 {
			start = 0
		}
		return buildWindow(full, start, width)
	}
	return tailVis, tailStart
}

// highlight using rune-count mapping from full → visible
func highlightInWindow(visible string, visStart int, occ *[2]int, full string, useColor bool) string {
	if !useColor || occ == nil {
		return visible
	}
	iFull, jFull := occ[0], occ[1]

	preRunes := utf8.RuneCountInString(full[visStart:iFull])
	midRunes := utf8.RuneCountInString(full[iFull:jFull])

	visRunes := []rune(visible)
	iVis := preRunes
	jVis := preRunes + midRunes
	if iVis < 0 || jVis > len(visRunes) || iVis >= jVis {
		return visible
	}
	return string(visRunes[:iVis]) + colorize(string(visRunes[iVis:jVis]), fgExe, useColor) + string(visRunes[jVis:])
}

// ---------- Rendering ----------
func printTree(children map[int][]int, meta map[int]Proc, roots []int, visible map[int]struct{}, a Args, pid2comm map[int]string) {
	termW := getTerminalWidth()
	header := fmt.Sprintf("%5s %5s %5s %*s  %s", "PID", "%CPU", "%MEM", userColWidth, "USER", "COMMAND")
	fmt.Println(header)

	useColor := isTTY()

	var walk func(pid int, prefix []string)
	walk = func(pid int, prefix []string) {
		if _, ok := visible[pid]; !ok {
			return
		}
		v := meta[pid]

		pidCol := padLeft(strconv.Itoa(v.PID), 5)
		cpuCol := padLeft(v.CPU, 5)
		memCol := padLeft(v.MEM, 5)
		userRaw := truncUser(v.User, userColWidth)
		userCol := padLeft(userRaw, userColWidth)

		leftPlain := fmt.Sprintf("%s %s %s %s  ", pidCol, cpuCol, memCol, userCol)
		treePlain := strings.Join(prefix, "")

		leftLen := displayWidth(leftPlain) + displayWidth(treePlain)
		maxCmd := termW - leftLen - 1
		if maxCmd < 10 {
			maxCmd = 10
		}

		full := v.Cmd
		comm := pid2comm[v.PID]

		target, occ := chooseTargetAndOcc(full, comm)

		var vis string
		var visStart int
		if a.Wide {
			vis, visStart = full, 0
		} else {
			if occ != nil {
				vis, visStart = tailOrAround(full, maxCmd, occ[0], occ[1], true)
			} else {
				vis, visStart = tailOrAround(full, maxCmd, 0, 0, false)
			}
		}

		_ = target // currently unused in output, but could be printed with a prefix if wanted

		leftCol := colorize(pidCol, fgPID, useColor) + " " + cpuCol + " " + memCol + " " +
			colorize(userCol, fgUser, useColor) + "  "
		treeCol := colorize(treePlain, fgTree, useColor)

		visColored := highlightInWindow(vis, visStart, occ, full, useColor)

		sep := ""
		if treeCol != "" {
			sep = " "
		}
		fmt.Print(leftCol + treeCol + sep + visColored + "\n")

		kids := []int{}
		for _, k := range children[pid] {
			if _, ok := visible[k]; ok {
				kids = append(kids, k)
			}
		}
		for i, k := range kids {
			last := i == len(kids)-1
			branch := "├──"
			if last {
				branch = "└──"
			}
			walk(k, append(prefix, branch))
		}
	}

	for _, r := range roots {
		if _, ok := visible[r]; ok {
			walk(r, []string{""})
		}
	}
}

func getTerminalWidth() int {
	w, ok := ttyWidth()
	if ok && w >= 40 {
		return w
	}
	if c := os.Getenv("COLUMNS"); c != "" {
		if n, err := strconv.Atoi(c); err == nil && n >= 40 {
			return n
		}
	}
	return 120
}

type winsize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

func ttyWidth() (int, bool) {
	// use ioctl(TIOCGWINSZ)
	const TIOCGWINSZ = 0x40087468 // from sys/ioctl.h on darwin
	ws := &winsize{}
	// FD 1 = stdout
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(syscall.Stdout), uintptr(TIOCGWINSZ), uintptr(unsafe.Pointer(ws)))
	if errno != 0 || ws.Col == 0 {
		// try stderr as fallback (some setups only attach TTY on 2)
		_, _, errno2 := syscall.Syscall(syscall.SYS_IOCTL, uintptr(syscall.Stderr), uintptr(TIOCGWINSZ), uintptr(unsafe.Pointer(ws)))
		if errno2 != 0 || ws.Col == 0 {
			return 0, false
		}
	}
	return int(ws.Col), true
}
