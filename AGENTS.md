<!-- SPDX-License-Identifier: Apache-2.0 -->
# Instructions

Read `README.md` first.

The coding standard is the `ai-coding-sop` repository at
`https://github.com/filmil/ai-coding-sop`.
Its `AGENTS.md`, its `prose-readability` skill, its `git-commit-rules`
skill and its `hermetic`, `bazel` and `bazel-bzl-files` fragments apply
here in full.
Where this file and ai-coding-sop differ, ai-coding-sop wins; ask the
operator about a conflict that matters.

# Agent operating rules

A change is complete only when `bazel test //...` passes, `bazel run
//tools:lint` is clean, the integration module passes (`cd integration &&
bazel test //... && bazel run //:lint`), and new tests cover the change.

1. Edit text sources only: Go sources, `BUILD.bazel`, `MODULE.bazel`,
   `.bzl`, and the lint's fixtures.
2. Keep every action hermetic. No host tools, no host files outside the
   workspace, no host environment variables, no network in actions.
3. Run `bazel run //tools:buildifier` and gofmt on every change.
4. Every check of the lint has a known-bad fixture under
   `tools/lint/testdata/bad` that it must fail on, with exactly its findings
   listed in `lint_test.go`, and known-good fixtures it must pass.
5. One logical change per pull request: one tool, one feature, or one fix.
6. The hdlf_rules modules use these tools as they are released. A change
   that makes a module's lint fail where it passed before says so in the
   pull request, with the modules it affects.
7. Stop and ask the operator before you:
   * delete a test or a known-bad fixture;
   * change a toolchain version or a library version;
   * run a release or publish workflow, or set a secret.
8. When a test fails and the cause is not clear after three attempts, stop.
   Write the failure and your hypotheses in the pull request.

# Rules particular to this repository

* **Development tools only.** A tool here is used by the hdlf_rules
  modules' own builds, tests and CI, never by their users. Runtime
  helpers that users load stay in each module.
* **Go only.** Every tool is pure Go (`pure = "on"`), so no module that
  uses it needs a C++ toolchain.
* **Linux x86_64 only** in v1, like the modules that use it.

`CLAUDE.md` and `GEMINI.md` are symlinks to this file.
