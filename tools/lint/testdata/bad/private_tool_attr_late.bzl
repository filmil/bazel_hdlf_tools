my_rule = rule(
    implementation = _impl,
    attrs = {
        "_tool": attr.label(
            executable = True,
            cfg = "exec",
            default = "@some_tools//:cli",
        ),
        "_doc_first": attr.label(doc = "a tool", default = "@other//:x"),
    },
)
