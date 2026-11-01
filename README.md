<!-- SPDX-License-Identifier: Apache-2.0 -->
[![Test](https://github.com/filmil/bazel_hdlf_tools/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/filmil/bazel_hdlf_tools/actions/workflows/test.yml)
[![Tag and Release](https://github.com/filmil/bazel_hdlf_tools/actions/workflows/tag-and-release.yml/badge.svg)](https://github.com/filmil/bazel_hdlf_tools/releases)
[![Publish to my Bazel registry](https://github.com/filmil/bazel_hdlf_tools/actions/workflows/publish.yml/badge.svg)](https://github.com/filmil/bazel_hdlf_tools/actions/workflows/publish.yml)

# hdlf_tools

Development tools shared by the hdlf_rules modules
(`hdlf_rules_kicad`, `_netlist`, `_brep`, `_fab`, `_cam` and `_em`).
Each tool is written once here, and every module takes it as a dev dependency.
A user of an hdlf_rules module never sees hdlf_tools.

The runtime helpers that a module's users load (`run_tool`, `determinism_test`, the launcher) are not here.
They stay in each module.

## The hermeticity lint

`tools/lint` scans a module's BUILD files, `.bzl` files, `MODULE.bazel`, `.bazelrc` files and CI workflows for hermeticity escapes:

* `PATH` or `HOME` passed to actions from the command line (`--action_env`, `--host_action_env`, `--test_env`);
* `use_default_shell_env = True`;
* the tags and execution requirements `local` and `no-sandbox`, and `local = True`;
* the exception tags `requires-network`, `requires-root` and `requires-fakeroot` on a target that the module's allowlist does not list;
* an action started outside the module's `run_tool.bzl`;
* a private attribute whose default names a tool in another repository, where tools must come from toolchains.

It walks the whole tree of the workspace it runs in, the integration module included.
No Bazel test can take a separate module's files as inputs, so the scan is a `bazel run` target, and a module's CI runs it on every pull request.

## Using it

`MODULE.bazel`:

```starlark
bazel_dep(name = "hdlf_tools", version = "0.0.1", dev_dependency = True)
```

`.bazelrc`, since the module is only in the filmil registry:

```
common --registry=https://raw.githubusercontent.com/filmil/bazel-registry/main
common --registry=https://bcr.bazel.build
```

There is no release yet; 0.0.1, above, will be the first.

`tools/BUILD.bazel`:

```starlark
load("@hdlf_tools//tools/lint:defs.bzl", "hermetic_lint")

hermetic_lint(
    name = "lint",
    # The one file allowed to start actions that run a tool; "none" if no
    # file may.
    run_tool = "em/private/run_tool.bzl",
    # The default; relative to the module root.
    allowlist = "tools/lint/hermetic_exceptions.txt",
)
```

Run it from anywhere in the module:

```sh
bazel run //tools:lint
```

It prints one line per finding, `file:line: [rule] message`, and exits 1 if there is any.

The allowlist has one line per target allowed to carry an exception tag, with the reason a person approved:

```
//path/to/package:target  needs the network to fetch X
```

A tag in a `.bzl` macro body is listed by the file, as `//pkg:file.bzl`.
A BUILD target whose name is computed (`name = "a" + "b"`) is listed by its BUILD file, as `//pkg:BUILD.bazel`.

`integration/` is a small module laid out like an hdlf_rules module, which uses the lint this way.

## Building

```sh
bazel test //...
bazel run //tools:lint
bazel run //tools:buildifier_check
cd integration && bazel test //... && bazel run //:lint
```

## Licenses

hdlf_tools is under the Apache License 2.0 ([LICENSE](LICENSE)).
The tools are Go, built with the Go SDK that rules_go downloads (BSD-3-Clause).
