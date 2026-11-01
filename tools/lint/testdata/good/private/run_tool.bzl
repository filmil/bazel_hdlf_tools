def run_tool(ctx, exe, outputs):
    ctx.actions.run(executable = exe, outputs = outputs)
