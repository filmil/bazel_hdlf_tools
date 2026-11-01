def fetching_test(name, **kwargs):
    native.sh_test(
        name = name,
        tags = ["requires-network"],
        **kwargs
    )
