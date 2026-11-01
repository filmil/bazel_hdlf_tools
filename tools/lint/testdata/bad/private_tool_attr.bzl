my_rule = rule(
    implementation = _impl,
    attrs = {
        "src": attr.label(),
        "_kicad_cli": attr.label(
            default = "@kicad_toolchains//:kicad_cli",
            executable = True,
            cfg = "exec",
        ),
        "_libs": attr.label_list(default = ["@@kicad_libs+//:standard"]),
    },
)
