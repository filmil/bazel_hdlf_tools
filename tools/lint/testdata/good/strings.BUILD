genrule(
    name = "quoted",
    outs = ["out.txt"],
    cmd = """
        # use_default_shell_env = True is only text here ( [ {
        echo '"local": 1' "tags = [\"no-sandbox\"]" > $@
    """,
    tags = ["manual"],
)

sh_test(
    name = 'allowed_root',
    srcs = ["t.sh"],
    tags = ["requires-root"],
)

sh_test(
    name = "allowed_fakeroot",
    srcs = ["t.sh"],
    tags = ["requires-fakeroot"],
)
