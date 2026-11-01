COMMON = ["manual"]

sh_test(
    name = 'concat',
    srcs = ["t.sh"],
    tags = COMMON + select({
        "//conditions:default": ["no-sandbox"],
    }),
)
