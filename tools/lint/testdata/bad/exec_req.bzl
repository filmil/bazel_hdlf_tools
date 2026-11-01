def _impl(ctx):
    ctx.actions.run(
        outputs = [ctx.outputs.out],
        executable = ctx.executable.tool,
        execution_requirements = {
            "no-sandbox": "1",
        },
    )
