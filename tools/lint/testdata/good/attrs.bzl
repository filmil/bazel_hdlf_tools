# Private attributes that name this repository or Bazel's own are fine, and
# so are actions that run no tool.
my_rule = rule(
    implementation = _impl,
    attrs = {
        "_template": attr.label(default = "//em/private:template.txt", allow_single_file = True),
        "_platform": attr.label(default = "@platforms//os:linux"),
        "_name": attr.string(default = "@not_a_label"),
        "public": attr.label(default = "@some_repo//:x"),
    },
)

def _impl(ctx):
    ctx.actions.write(ctx.outputs.out, "x")
    ctx.actions.symlink(output = ctx.outputs.link, target_file = ctx.file.src)
