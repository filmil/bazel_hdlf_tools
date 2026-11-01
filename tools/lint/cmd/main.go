// SPDX-License-Identifier: Apache-2.0

// Command hermetic_lint scans the source tree for hermeticity escapes.
//
// Run it from the repository with:
//
//	bazel run //tools/lint
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
		"allowlist of exception tags, relative to the root")
	flag.Parse()
	if *root == "" {
		fmt.Fprintln(os.Stderr, "hermetic_lint: no root; run it with `bazel run //tools/lint` or pass --root")
		os.Exit(2)
	}
	data, err := os.ReadFile(filepath.Join(*root, *allowPath))
	if err != nil {
		fmt.Fprintln(os.Stderr, "hermetic_lint:", err)
		os.Exit(2)
	}
	allow, err := lint.ParseAllowlist(data)
	if err != nil {
		fmt.Fprintln(os.Stderr, "hermetic_lint:", err)
		os.Exit(2)
	}
	findings, err := lint.Walk(*root, allow)
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
