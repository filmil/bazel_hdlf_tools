# A doc string that starts with "@" is not a default.
my_rule = rule(
    implementation = _impl,
    attrs = {
        "_os": attr.label(
            doc = "@platforms constraint; the default is ours.",
            default = "//x:os",
        ),
    },
)
