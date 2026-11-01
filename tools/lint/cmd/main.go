// SPDX-License-Identifier: Apache-2.0

// Command hermetic_lint scans the source tree for hermeticity escapes.
//
// A module runs it through the hermetic_lint rule in
// @hdlf_tools//tools/lint:defs.bzl, which passes the module's settings:
//
//	bazel run //tools:lint
//
// It walks the whole tree, the integration module included, which no Bazel
// test can see as inputs. It prints one line per finding and exits 1 if
// there is any.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/filmil/bazel_hdlf_tools/tools/lint"
)

func main() {
	os.Exit(run(os.Args[1:], os.Getenv("BUILD_WORKSPACE_DIRECTORY"), os.Stdout, os.Stderr))
}

// run is the command, apart from the process: it returns the exit code, 0
// when the tree is clean, 1 when there are findings, 2 on a usage or read
// error.
func run(args []string, workspace string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("hermetic_lint", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", workspace,
		"repository root to scan (default: the workspace of `bazel run`)")
	allowPath := fs.String("allowlist", "tools/lint/hermetic_exceptions.txt",
		"allowlist of exception tags, relative to the root unless absolute")
	runTool := fs.String("run_tool", "",
		"the module's run_tool.bzl, relative to the root: the only file that may start actions; \"none\" if no file may (required)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *root == "" {
		fmt.Fprintln(stderr, "hermetic_lint: no root; run it with `bazel run` or pass --root")
		return 2
	}
	if *runTool == "" {
		fmt.Fprintln(stderr, "hermetic_lint: --run_tool is required, e.g. --run_tool=em/private/run_tool.bzl, or --run_tool=none")
		return 2
	}
	if *runTool == "none" {
		*runTool = ""
	}
	data, err := os.ReadFile(lint.Under(*root, *allowPath))
	if err != nil {
		fmt.Fprintln(stderr, "hermetic_lint:", err)
		return 2
	}
	allow, err := lint.ParseAllowlist(data)
	if err != nil {
		fmt.Fprintln(stderr, "hermetic_lint:", err)
		return 2
	}
	findings, err := lint.Walk(*root, lint.Config{Allow: allow, AllowlistPath: *allowPath, RunTool: filepath.ToSlash(*runTool)})
	if err != nil {
		fmt.Fprintln(stderr, "hermetic_lint:", err)
		return 2
	}
	for _, f := range findings {
		fmt.Fprintln(stdout, f)
	}
	if len(findings) > 0 {
		fmt.Fprintf(stderr, "hermetic_lint: %d finding(s)\n", len(findings))
		return 1
	}
	fmt.Fprintln(stdout, "hermetic_lint: clean")
	return 0
}
