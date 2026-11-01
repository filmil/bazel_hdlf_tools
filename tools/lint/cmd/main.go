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
	"os"
	"path/filepath"

	"github.com/filmil/bazel_hdlf_tools/tools/lint"
)

func main() {
	root := flag.String("root", os.Getenv("BUILD_WORKSPACE_DIRECTORY"),
		"repository root to scan (default: the workspace of `bazel run`)")
	allowPath := flag.String("allowlist", "tools/lint/hermetic_exceptions.txt",
		"allowlist of exception tags, relative to the root unless absolute")
	runTool := flag.String("run_tool", "",
		"the module's run_tool.bzl, relative to the root: the only file that may start actions; \"none\" if no file may (required)")
	flag.Parse()
	if *runTool == "" {
		fmt.Fprintln(os.Stderr, "hermetic_lint: --run_tool is required, e.g. --run_tool=em/private/run_tool.bzl, or --run_tool=none")
		os.Exit(2)
	}
	if *runTool == "none" {
		*runTool = ""
	}
	if *root == "" {
		fmt.Fprintln(os.Stderr, "hermetic_lint: no root; run it with `bazel run` or pass --root")
		os.Exit(2)
	}
	data, err := os.ReadFile(lint.Under(*root, *allowPath))
	if err != nil {
		fmt.Fprintln(os.Stderr, "hermetic_lint:", err)
		os.Exit(2)
	}
	allow, err := lint.ParseAllowlist(data)
	if err != nil {
		fmt.Fprintln(os.Stderr, "hermetic_lint:", err)
		os.Exit(2)
	}
	findings, err := lint.Walk(*root, lint.Config{Allow: allow, RunTool: filepath.ToSlash(*runTool)})
	if err != nil {
		fmt.Fprintln(os.Stderr, "hermetic_lint:", err)
		os.Exit(2)
	}
	for _, f := range findings {
		fmt.Println(f)
	}
	if len(findings) > 0 {
		fmt.Fprintf(os.Stderr, "hermetic_lint: %d finding(s)\n", len(findings))
		os.Exit(1)
	}
	fmt.Println("hermetic_lint: clean")
}
