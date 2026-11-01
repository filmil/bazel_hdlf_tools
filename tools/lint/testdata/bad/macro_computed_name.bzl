def network_test(name):
    native.sh_test(
        name = name + "_test",
        tags = ["requires-network"],
    )
