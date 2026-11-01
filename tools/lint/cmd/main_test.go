// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// tree writes files under a new directory and returns it.
func tree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

const allowlist = "# none\n"

func runLint(t *testing.T, root string, args ...string) (int, string) {
	t.Helper()
	var out, errs bytes.Buffer
	code := run(args, root, &out, &errs)
	return code, out.String() + errs.String()
}

func TestCleanTreeExitsZero(t *testing.T) {
	root := tree(t, map[string]string{
		"tools/lint/hermetic_exceptions.txt": allowlist,
		"m/private/run_tool.bzl":             "def f(ctx):\n    ctx.actions.run(executable = ctx.executable.x)\n",
	})
	if code, out := runLint(t, root, "--run_tool=m/private/run_tool.bzl"); code != 0 || !strings.Contains(out, "clean") {
		t.Errorf("want exit 0 and clean, got %d: %s", code, out)
	}
}

func TestFindingsExitOne(t *testing.T) {
	root := tree(t, map[string]string{
		"tools/lint/hermetic_exceptions.txt": allowlist,
		"BUILD.bazel":                        "genrule(name = \"a\", tags = [\"local\"])\n",
	})
	code, out := runLint(t, root, "--run_tool=none")
	if code != 1 || !strings.Contains(out, "BUILD.bazel:1: [forbidden-tag]") {
		t.Errorf("want exit 1 and the finding, got %d: %s", code, out)
	}
}

// --run_tool=none: no file may start actions, not even a run_tool.bzl.
func TestRunToolNone(t *testing.T) {
	root := tree(t, map[string]string{
		"tools/lint/hermetic_exceptions.txt": allowlist,
		"m/private/run_tool.bzl":             "def f(ctx):\n    ctx.actions.run(executable = ctx.executable.x)\n",
	})
	code, out := runLint(t, root, "--run_tool=none")
	if code != 1 || !strings.Contains(out, "this module starts no actions") {
		t.Errorf("want exit 1 and action-outside-run-tool, got %d: %s", code, out)
	}
}

func TestRunToolRequired(t *testing.T) {
	root := tree(t, map[string]string{"tools/lint/hermetic_exceptions.txt": allowlist})
	if code, out := runLint(t, root); code != 2 || !strings.Contains(out, "--run_tool is required") {
		t.Errorf("want exit 2 and the usage error, got %d: %s", code, out)
	}
}

// An absolute --allowlist is read as given, and the message names it.
func TestAbsoluteAllowlist(t *testing.T) {
	root := tree(t, map[string]string{
		"BUILD.bazel": "sh_test(name = \"t\", tags = [\"requires-network\"])\n",
	})
	elsewhere := tree(t, map[string]string{"allow.txt": "//:t  approved for the test\n"})
	abs := filepath.Join(elsewhere, "allow.txt")
	if code, out := runLint(t, root, "--run_tool=none", "--allowlist="+abs); code != 0 {
		t.Errorf("allowlisted by an absolute path: want exit 0, got %d: %s", code, out)
	}
	empty := tree(t, map[string]string{"allow.txt": allowlist})
	code, out := runLint(t, root, "--run_tool=none", "--allowlist="+filepath.Join(empty, "allow.txt"))
	if code != 1 || !strings.Contains(out, filepath.Join(empty, "allow.txt")) {
		t.Errorf("want exit 1 naming the allowlist, got %d: %s", code, out)
	}
}

func TestMissingAllowlistExitsTwo(t *testing.T) {
	if code, _ := runLint(t, t.TempDir(), "--run_tool=none"); code != 2 {
		t.Errorf("want exit 2, got %d", code)
	}
}
