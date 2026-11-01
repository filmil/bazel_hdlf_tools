def _impl(ctx):
    ctx.actions.run_shell(
        outputs = [ctx.outputs.out],
        command = "kicad-cli version > $1",
        use_default_shell_env = True,
    )
