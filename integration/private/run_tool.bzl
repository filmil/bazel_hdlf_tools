# SPDX-License-Identifier: Apache-2.0
"""The one file of this module that starts actions that run a tool."""

def run_tool(ctx, tool, args, outputs):
    ctx.actions.run(executable = tool, arguments = args, outputs = outputs)
