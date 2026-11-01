// SPDX-License-Identifier: Apache-2.0

// Package lint finds hermeticity escapes in Bazel files.
//
// It reads BUILD files, .bzl files, MODULE.bazel, .bazelrc files and CI
// workflows, and reports each use of a forbidden flag, attribute or tag.
// The rules come from the specification "Hermetic hardware builds with
// Bazel": actions must not take PATH or HOME from the command line or the
// host, must not run outside the sandbox, and must not use the network
// unless a person approved an exception.
package lint

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Finding is one forbidden use, at a file and line.
type Finding struct {
	File string
	Line int
	Rule string
	Msg  string
}

func (f Finding) String() string {
	return fmt.Sprintf("%s:%d: [%s] %s", f.File, f.Line, f.Rule, f.Msg)
}

// Kind says how a file is read.
type Kind int

const (
	// KindNone is a file the lint does not read.
	KindNone Kind = iota
	// KindStarlark is a BUILD, .bzl or MODULE.bazel file.
	KindStarlark
	// KindRC is a .bazelrc file.
	KindRC
	// KindWorkflow is a CI workflow.
	KindWorkflow
)

// KindOf returns how the lint reads the file at path. Test fixtures use the
// suffixes .BUILD and .bazelrc so that Bazel does not take them for
// packages or settings.
func KindOf(path string) Kind {
	base := filepath.Base(path)
	switch {
	case base == "BUILD" || base == "BUILD.bazel" || base == "MODULE.bazel" ||
		strings.HasSuffix(base, ".BUILD") || strings.HasSuffix(base, ".bzl"):
		return KindStarlark
	case base == ".bazelrc" || strings.HasSuffix(base, ".bazelrc"):
		return KindRC
	case (strings.HasSuffix(base, ".yml") || strings.HasSuffix(base, ".yaml")) &&
		strings.Contains(filepath.ToSlash(path), "workflows/"):
		return KindWorkflow
	}
	return KindNone
}

// ForbiddenTags may never appear, as a tag or as an execution requirement.
var ForbiddenTags = []string{"local", "no-sandbox"}

// ExceptionTags may appear, as a tag or as an execution requirement, only
// on targets listed in the allowlist.
var ExceptionTags = []string{"requires-network", "requires-root", "requires-fakeroot"}

var (
	// --action_env=PATH, --action_env PATH, --action_env=HOME=/x, and the
	// same for --host_action_env and --test_env. A fixed value is refused
	// too: it names a host directory.
	reHostEnv = regexp.MustCompile(`--(?:host_)?(?:action|test)_env[= ](PATH|HOME)\b`)
	// These match the masked text, where string contents are blanked.
	reShellEnv = regexp.MustCompile(`\buse_default_shell_env\s*=\s*True\b`)
	reLocal    = regexp.MustCompile(`\blocal\s*=\s*(?:True|1)\b`)
	reTags     = regexp.MustCompile(`\b(?:tags|execution_requirements)\s*=[^=]`)
	reNameAttr = regexp.MustCompile(`\bname\s*=[^=]`)
	reActions  = regexp.MustCompile(`\bactions\.run(?:_shell)?\s*\(`)
	reAttrLbl  = regexp.MustCompile(`\battr\.label(?:_list|_keyed_string_dict)?\s*\(`)
)

// Allowlist maps a target, as "path/to/package:name", to the reason a person
// gave for its exception tag. A tag in a .bzl macro body is listed by the
// file, as "path/to/package:file.bzl".
type Allowlist map[string]string

// ParseAllowlist reads lines of the form "//pkg:name  reason". Blank lines
// and lines starting with # are skipped. A line without a reason is an
// error: the reason is what the person approved.
func ParseAllowlist(data []byte) (Allowlist, error) {
	a := Allowlist{}
	s := bufio.NewScanner(bytes.NewReader(data))
	n := 0
	for s.Scan() {
		n++
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Fields(line)
		if len(f) < 2 || !strings.HasPrefix(f[0], "//") || !strings.Contains(f[0], ":") {
			return nil, fmt.Errorf("allowlist line %d: want \"//pkg:name reason\", got %q", n, line)
		}
		a[strings.TrimPrefix(f[0], "//")] = strings.Join(f[1:], " ")
	}
	return a, s.Err()
}

// Scan returns the findings for one file. pkg is the file's Bazel package
// ("" for the root, "integration/foo" inside the integration module), used
// to look targets up in the allowlist.
func Scan(path, pkg string, data []byte, allow Allowlist) []Finding {
	var out []Finding
	switch KindOf(path) {
	case KindStarlark:
		out = scanStarlark(path, pkg, string(data), allow)
	case KindRC, KindWorkflow:
		out = scanFlags(path, string(data))
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Line < out[j].Line })
	return out
}

func hostEnvMsg(v string) string {
	return fmt.Sprintf("sets %s for actions from the command line; tools come from toolchains and run_tool sets HOME", v)
}

func scanFlags(path, src string) []Finding {
	var out []Finding
	for i, line := range strings.Split(src, "\n") {
		if j := strings.Index(line, "#"); j >= 0 {
			line = line[:j]
		}
		for _, m := range reHostEnv.FindAllStringSubmatch(line, -1) {
			out = append(out, Finding{path, i + 1, "host-env", hostEnvMsg(m[1])})
		}
	}
	return out
}

// str is one string literal: its value and where it is in the source.
type str struct {
	start, end int // byte offsets of the whole literal, quotes included
	val        string
}

// tokenize finds the string literals and comments of a Starlark file. It
// returns the source with comments and string contents blanked (quotes and
// line breaks kept, so offsets and line numbers stay right), and the
// literals in order. Triple-quoted strings and escapes are handled; string
// prefixes such as r or b stay in the masked text as letters.
func tokenize(src string) (string, []str) {
	m := []byte(src)
	var strs []str
	for i := 0; i < len(src); {
		c := src[i]
		switch {
		case c == '#':
			for i < len(src) && src[i] != '\n' {
				m[i] = ' '
				i++
			}
		case c == '"' || c == '\'':
			q := string(c)
			if strings.HasPrefix(src[i:], q+q+q) {
				q = q + q + q
			}
			start := i
			i += len(q)
			var val strings.Builder
			for i < len(src) {
				if src[i] == '\\' && i+1 < len(src) {
					val.WriteByte(src[i])
					val.WriteByte(src[i+1])
					blank(m, i, i+2)
					i += 2
					continue
				}
				if strings.HasPrefix(src[i:], q) {
					break
				}
				if len(q) == 1 && src[i] == '\n' {
					break // unterminated; stop at the line end
				}
				val.WriteByte(src[i])
				blank(m, i, i+1)
				i++
			}
			if strings.HasPrefix(src[i:], q) {
				i += len(q)
			}
			strs = append(strs, str{start, i, val.String()})
		default:
			i++
		}
	}
	return string(m), strs
}

func blank(m []byte, from, to int) {
	for k := from; k < to; k++ {
		if m[k] != '\n' {
			m[k] = ' '
		}
	}
}

// depths returns, for each byte of the masked text, the bracket depth just
// before it.
func depths(masked string) []int {
	d := make([]int, len(masked)+1)
	cur := 0
	for i := 0; i < len(masked); i++ {
		d[i] = cur
		switch masked[i] {
		case '(', '[', '{':
			cur++
		case ')', ']', '}':
			cur--
		}
	}
	d[len(masked)] = cur
	return d
}

// exprEnd returns where the expression starting at from ends: at a comma or
// a closing bracket at the starting depth, or at a line end at depth 0.
func exprEnd(masked string, d []int, from int) int {
	base := d[from]
	for i := from; i < len(masked); i++ {
		switch c := masked[i]; {
		case d[i] == base && (c == ',' || c == ')' || c == ']' || c == '}'):
			return i
		case d[i] < base:
			return i
		case base == 0 && c == '\n' && d[i] == 0:
			return i
		}
	}
	return len(masked)
}

type span struct {
	start, end int
	name       string
}

// topCalls returns each top-level call (a rule or macro use) with its name
// attribute. A def signature is not a call.
func topCalls(masked string, d []int, strs []str) []span {
	var out []span
	for i := 0; i < len(masked); i++ {
		if masked[i] != '(' || d[i] != 0 {
			continue
		}
		ls := strings.LastIndexByte(masked[:i], '\n') + 1
		if strings.HasPrefix(strings.TrimSpace(masked[ls:i]), "def ") {
			continue
		}
		end := i + 1
		for end < len(masked) && d[end+1] > 0 {
			end++
		}
		s := span{start: i, end: end}
		for _, loc := range reNameAttr.FindAllStringIndex(masked[i:end], -1) {
			if d[i+loc[0]] != 1 {
				continue // a name inside a nested call
			}
			from := i + loc[1] - 1
			to := exprEnd(masked, d, from)
			for _, t := range strs {
				if t.start >= from && t.start < to {
					s.name = t.val
					break
				}
			}
			break
		}
		out = append(out, s)
		i = end
	}
	return out
}

func scanStarlark(path, pkg, src string, allow Allowlist) []Finding {
	masked, strs := tokenize(src)
	d := depths(masked)
	lineAt := lineIndex(src)
	var out []Finding

	for _, loc := range reShellEnv.FindAllStringIndex(masked, -1) {
		out = append(out, Finding{path, lineAt(loc[0]), "shell-env",
			"use_default_shell_env = True gives actions the host environment"})
	}
	for _, loc := range reLocal.FindAllStringIndex(masked, -1) {
		out = append(out, Finding{path, lineAt(loc[0]), "forbidden-tag",
			"local = True runs the action outside the sandbox"})
	}
	for _, t := range strs {
		for _, m := range reHostEnv.FindAllStringSubmatch(t.val, -1) {
			out = append(out, Finding{path, lineAt(t.start), "host-env", hostEnvMsg(m[1])})
		}
	}

	// Every action runs its tool through run_tool (spec "Toolchains",
	// requirement 3), which lives in a file named private/run_tool.bzl.
	if !strings.HasSuffix(filepath.ToSlash(path), RunToolFile) {
		for _, loc := range reActions.FindAllStringIndex(masked, -1) {
			out = append(out, Finding{path, lineAt(loc[0]), "action-outside-run-tool",
				"start actions that run a tool through run_tool, in " + RunToolFile})
		}
	}

	// A private attribute must not name a tool in another repository
	// (requirement 2): tools come from ctx.toolchains.
	for _, t := range strs {
		rest := strings.TrimLeft(masked[t.end:], " \t")
		if !strings.HasPrefix(t.val, "_") || !strings.HasPrefix(rest, ":") {
			continue
		}
		from := t.end + (len(masked[t.end:]) - len(rest)) + 1
		to := exprEnd(masked, d, from)
		if !reAttrLbl.MatchString(masked[from:to]) {
			continue
		}
		for _, v := range strs {
			if v.start >= from && v.start < to && external(v.val) {
				out = append(out, Finding{path, lineAt(t.start), "private-tool-attr",
					fmt.Sprintf("private attribute %q names %q; get tools from ctx.toolchains", t.val, v.val)})
				break
			}
		}
	}

	// Values that act as tags: every string in a tags or
	// execution_requirements expression (a list, a concatenation, a select,
	// a dict), and every string used as a dict key anywhere, which covers
	// execution requirements built apart from the call.
	var uses []str
	seen := map[int]bool{}
	for _, loc := range reTags.FindAllStringIndex(masked, -1) {
		from := loc[1] - 1
		to := exprEnd(masked, d, from)
		for _, t := range strs {
			if t.start >= from && t.start < to && !seen[t.start] {
				seen[t.start] = true
				uses = append(uses, t)
			}
		}
	}
	for _, t := range strs {
		rest := strings.TrimLeft(masked[t.end:], " \t")
		if strings.HasPrefix(rest, ":") && !seen[t.start] {
			seen[t.start] = true
			uses = append(uses, t)
		}
	}

	calls := topCalls(masked, d, strs)
	label := func(pos int) string {
		for _, c := range calls {
			if pos > c.start && pos < c.end && c.name != "" {
				return pkg + ":" + c.name
			}
		}
		return pkg + ":" + filepath.Base(path)
	}
	for _, u := range uses {
		for _, bad := range ForbiddenTags {
			if u.val == bad {
				out = append(out, Finding{path, lineAt(u.start), "forbidden-tag",
					fmt.Sprintf("%q runs the action outside the sandbox", bad)})
			}
		}
		for _, ex := range ExceptionTags {
			if u.val != ex {
				continue
			}
			l := label(u.start)
			if _, ok := allow[l]; !ok {
				out = append(out, Finding{path, lineAt(u.start), "exception-tag",
					fmt.Sprintf("%q on //%s needs a line in tools/lint/hermetic_exceptions.txt, approved by a person", ex, l)})
			}
		}
	}
	return out
}

// RunToolFile is the only file allowed to start actions that run tools.
const RunToolFile = "private/run_tool.bzl"

// external reports whether a label names another repository. Bazel's own
// repositories are fine.
func external(label string) bool {
	if !strings.HasPrefix(label, "@") {
		return false
	}
	for _, ok := range []string{"@bazel_tools//", "@platforms//", "@@bazel_tools//", "@@platforms//"} {
		if strings.HasPrefix(label, ok) {
			return false
		}
	}
	return true
}

// lineIndex returns a function from byte offset to 1-based line number.
func lineIndex(src string) func(int) int {
	starts := []int{0}
	for i := 0; i < len(src); i++ {
		if src[i] == '\n' {
			starts = append(starts, i+1)
		}
	}
	return func(off int) int {
		return sort.Search(len(starts), func(i int) bool { return starts[i] > off })
	}
}

// FixtureDir holds the lint's own known-bad files; Walk skips it.
const FixtureDir = "tools/lint/testdata"

// Walk scans every file under root that the lint reads, and returns the
// findings with paths relative to root. It skips .git, Bazel's output
// symlinks and FixtureDir. Packages inside the integration module are
// looked up in the allowlist as "integration/...".
func Walk(root string, allow Allowlist) ([]Finding, error) {
	var out []Finding
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			name := d.Name()
			if p != root && (name == ".git" || strings.HasPrefix(name, "bazel-") || rel == FixtureDir) {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 || KindOf(p) == KindNone {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		pkg := filepath.ToSlash(filepath.Dir(rel))
		if pkg == "." {
			pkg = ""
		}
		out = append(out, Scan(rel, pkg, data, allow)...)
		return nil
	})
	return out, err
}
