# A doc string that starts with "@" is not a default, wherever it is.
my_rule = rule(
    implementation = _impl,
    attrs = {
        "_os": attr.label(
            doc = "@platforms constraint; the default is ours.",
            default = "//x:os",
        ),
        "_os2": attr.label(
            default = "//x:os",
            doc = "@platforms again, after the default.",
        ),
    },
)
