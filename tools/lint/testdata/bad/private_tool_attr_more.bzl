_EXT_TOOL = "@some_tools//:cli"

my_rule = rule(
    implementation = _impl,
    attrs = {
        "_from_const": attr.label(default = _EXT_TOOL, executable = True, cfg = "exec"),
        "_two": attr.label_list(default = ["@a//:x", "@b//:y"]),
        "_split"
        : attr.label(default = "@c//:z"),
    },
)
