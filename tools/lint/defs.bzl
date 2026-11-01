# SPDX-License-Identifier: Apache-2.0
"""The hermeticity lint, as a target a module runs on its own tree."""

def _hermetic_lint_impl(ctx):
    # Not the bare name: //tools:lint would collide with a tools/lint package.
    out = ctx.actions.declare_file(ctx.label.name + "_hermetic_lint")
    ctx.actions.symlink(output = out, target_file = ctx.executable._lint, is_executable = True)
    return [DefaultInfo(executable = out, runfiles = ctx.runfiles(files = [ctx.executable._lint]))]

_hermetic_lint = rule(
    implementation = _hermetic_lint_impl,
    executable = True,
    attrs = {
        "_lint": attr.label(
            default = Label("//tools/lint"),
            executable = True,
            cfg = "target",
        ),
    },
)

def hermetic_lint(name, run_tool, allowlist = "tools/lint/hermetic_exceptions.txt", **kwargs):
    """Declares `bazel run` target that scans the module for hermeticity escapes.

    It walks the whole source tree of the workspace it runs in, the
    integration module included, which no Bazel test can take as inputs. It
    prints one line per finding and exits 1 if there is any. A module's CI
    runs it on every pull request.

    Example, in a module's `tools/BUILD.bazel`:

        load("@hdlf_tools//tools/lint:defs.bzl", "hermetic_lint")

        hermetic_lint(
            name = "lint",
            run_tool = "em/private/run_tool.bzl",
        )

    Args:
      name: the target's name.
      run_tool: the module's run_tool.bzl, relative to the module root: the
        only file allowed to start actions that run a tool. "none" if no
        file may.
      allowlist: the allowlist of exception tags, relative to the module
        root.
      **kwargs: other attributes, such as visibility.
    """
    _hermetic_lint(
        name = name,
        args = [
            "--run_tool=" + run_tool,
            "--allowlist=" + allowlist,
        ],
        **kwargs
    )
