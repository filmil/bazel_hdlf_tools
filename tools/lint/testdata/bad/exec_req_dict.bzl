_REQS = {"requires-fakeroot": "1", "local": "1"}

def _impl(ctx):
    ctx.actions.run(
        outputs = [ctx.outputs.out],
        executable = ctx.executable.tool,
        execution_requirements = _REQS,
    )
