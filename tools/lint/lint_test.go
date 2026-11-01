// SPDX-License-Identifier: Apache-2.0

package lint

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// at is an expected finding: its rule and line.
type at struct {
	rule string
	line int
}

func rulesAndLines(fs []Finding) []at {
	out := []at{}
	for _, f := range fs {
		out = append(out, at{f.Rule, f.Line})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].line != out[j].line {
			return out[i].line < out[j].line
		}
		return out[i].rule < out[j].rule
	})
	return out
}

// badCases lists every known-bad fixture with exactly the findings it must
// give (spec T3: each check fails on its known-bad input). Extra findings
// fail the test as well as missing ones.
var badCases = map[string][]at{
	"testdata/bad/actions_run.bzl":         {{"action-outside-run-tool", 2}, {"action-outside-run-tool", 7}},
	"testdata/bad/private_tool_attr.bzl":   {{"private-tool-attr", 5}, {"private-tool-attr", 10}},
	"testdata/bad/shell_env.bzl":           {{"action-outside-run-tool", 2}, {"shell-env", 5}},
	"testdata/bad/local_tag.BUILD":         {{"forbidden-tag", 7}},
	"testdata/bad/no_sandbox_tag.BUILD":    {{"forbidden-tag", 5}},
	"testdata/bad/local_attr.BUILD":        {{"forbidden-tag", 5}},
	"testdata/bad/concat_tags.BUILD":       {{"forbidden-tag", 7}},
	"testdata/bad/network_tag.BUILD":       {{"exception-tag", 4}},
	"testdata/bad/requires_root.BUILD":     {{"exception-tag", 1}},
	"testdata/bad/macro_network.bzl":       {{"exception-tag", 4}},
	"testdata/bad/exec_req.bzl":            {{"action-outside-run-tool", 2}, {"forbidden-tag", 6}},
	"testdata/bad/exec_req_dict.bzl":       {{"exception-tag", 1}, {"forbidden-tag", 1}, {"action-outside-run-tool", 4}},
	"testdata/bad/action_env.bazelrc":      {{"host-env", 1}},
	"testdata/bad/action_env_home.bazelrc": {{"host-env", 1}},
	"testdata/bad/two_on_a_line.bazelrc":   {{"host-env", 1}, {"host-env", 1}},
	"testdata/bad/workflows/ci.yml":        {{"host-env", 4}},
	"testdata/bad/module/MODULE.bazel":     {{"host-env", 4}},
}

func scanFile(t *testing.T, path, pkg string, allow Allowlist) []Finding {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return Scan(path, pkg, data, allow)
}

func TestBadFixturesFail(t *testing.T) {
	for file, want := range badCases {
		got := scanFile(t, file, "pkg", Allowlist{})
		if !reflect.DeepEqual(rulesAndLines(got), want) {
			t.Errorf("%s:\n want %v\n got  %v", file, want, got)
		}
	}
}

// Every file under testdata/bad has a case above, so a new fixture without
// a check cannot pass silently.
func TestEveryBadFixtureIsChecked(t *testing.T) {
	err := filepath.WalkDir("testdata/bad", func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			if _, ok := badCases[filepath.ToSlash(p)]; !ok {
				t.Errorf("%s has no case in badCases", p)
			}
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

// Strings and comments that mention forbidden things are not findings, and
// allowlisted exception tags (single-quoted names too) pass.
func TestGoodFixturesPass(t *testing.T) {
	allow := Allowlist{
		"good:allowed_network":  "a person approved it",
		"good:allowed_root":     "a person approved it",
		"good:allowed_fakeroot": "a person approved it",
	}
	for _, f := range []string{
		"testdata/good/clean.BUILD",
		"testdata/good/clean.bazelrc",
		"testdata/good/strings.BUILD",
		"testdata/good/private/run_tool.bzl",
		"testdata/good/attrs.bzl",
	} {
		if got := scanFile(t, f, "good", allow); len(got) != 0 {
			t.Errorf("%s: want no findings, got %v", f, got)
		}
	}
}

// An exception tag names the target in its message, and the allowlist
// matches that target only; a macro body is matched by its file.
func TestAllowlistLabels(t *testing.T) {
	cases := []struct {
		file, label string
	}{
		{"testdata/bad/network_tag.BUILD", "x:fetches_things"},
		{"testdata/bad/requires_root.BUILD", "x:as_root"},
		{"testdata/bad/macro_network.bzl", "x:macro_network.bzl"},
		{"testdata/bad/concat_tags.BUILD", ""},
	}
	for _, c := range cases {
		got := scanFile(t, c.file, "x", Allowlist{})
		if c.label == "" {
			continue
		}
		if len(got) != 1 || !strings.Contains(got[0].Msg, "//"+c.label) {
			t.Errorf("%s: want one finding naming //%s, got %v", c.file, c.label, got)
		}
		if got := scanFile(t, c.file, "x", Allowlist{c.label: "ok"}); len(got) != 0 {
			t.Errorf("%s with //%s allowlisted: want no findings, got %v", c.file, c.label, got)
		}
		if got := scanFile(t, c.file, "x", Allowlist{"x:other": "ok"}); len(got) != 1 {
			t.Errorf("%s with another target allowlisted: want 1 finding, got %v", c.file, got)
		}
	}
}

func TestParseAllowlist(t *testing.T) {
	a, err := ParseAllowlist([]byte("# c\n\n//a/b:c  needs the network for X\n"))
	if err != nil {
		t.Fatal(err)
	}
	if a["a/b:c"] != "needs the network for X" {
		t.Errorf("got %v", a)
	}
	for _, bad := range []string{"//a:b\n", "a:b reason\n", "//ab reason\n"} {
		if _, err := ParseAllowlist([]byte(bad)); err == nil {
			t.Errorf("%q: want an error", bad)
		}
	}
}

func TestTokenizeKeepsOffsets(t *testing.T) {
	src := "a = \"x # y\"  # z\nb = '''\n(\n'''\n"
	masked, strs := tokenize(src)
	if len(masked) != len(src) || strings.Count(masked, "\n") != strings.Count(src, "\n") {
		t.Fatalf("masked text changed length or lines: %q", masked)
	}
	if strings.ContainsAny(masked, "#(") {
		t.Errorf("comment or string content left in %q", masked)
	}
	var vals []string
	for _, s := range strs {
		vals = append(vals, s.val)
	}
	if fmt.Sprint(vals) != "[x # y \n(\n]" {
		t.Errorf("strings: got %q", vals)
	}
}

// Walk scans a tree, skips the lint's own fixtures and Bazel output links,
// and names files relative to the root.
func TestWalk(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("BUILD.bazel", "genrule(name = \"a\", tags = [\"local\"])\n")
	write("integration/BUILD.bazel", "sh_test(name = \"t\", tags = [\"requires-network\"])\n")
	write("tools/lint/testdata/BUILD.bazel", "genrule(name = \"skip\", tags = [\"local\"])\n")
	write("other/testdata/BUILD.bazel", "genrule(name = \"seen\", local = 1)\n")
	write(".bazelrc", "build --action_env=HOME\n")
	write("notes.txt", "use_default_shell_env = True\n")

	got, err := Walk(root, Allowlist{})
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for _, f := range got {
		lines = append(lines, f.String())
	}
	all := strings.Join(lines, "\n")
	for _, want := range []string{
		"BUILD.bazel:1: [forbidden-tag]",
		"integration/BUILD.bazel:1: [exception-tag]",
		"other/testdata/BUILD.bazel:1: [forbidden-tag]",
		".bazelrc:1: [host-env]",
	} {
		if !strings.Contains(all, want) {
			t.Errorf("want %q in:\n%s", want, all)
		}
	}
	if len(got) != 4 {
		t.Errorf("want 4 findings, got:\n%s", all)
	}

	got, err = Walk(root, Allowlist{"integration:t": "approved"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Errorf("with the integration test allowlisted: want 3 findings, got %v", got)
	}
}
