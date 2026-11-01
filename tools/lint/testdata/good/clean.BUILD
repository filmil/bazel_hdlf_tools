# A comment may say use_default_shell_env = True or tags = ["local"].
genrule(
    name = "fine",
    outs = ["out.txt"],
    cmd = "echo local > $@",
    tags = ["manual"],
)

sh_test(
    name = "allowed_network",
    srcs = ["t.sh"],
    tags = ["requires-network"],
)
