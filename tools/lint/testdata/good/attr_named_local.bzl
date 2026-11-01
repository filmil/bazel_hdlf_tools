# Attribute names that equal a tag are not tags.
my_rule = rule(
    implementation = _impl,
    attrs = {
        "local": attr.bool(default = False),
        "requires-network": attr.string(),
    },
)
