def _impl(ctx):
    ctx.actions.run(
        outputs = [ctx.outputs.out],
        executable = ctx.toolchains["//t:type"].tools["x"],
    )
    actions = ctx.actions
    actions.run_shell(outputs = [ctx.outputs.log], command = "true")
