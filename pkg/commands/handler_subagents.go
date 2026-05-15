package commands

import (
	"context"
	"fmt"
)

func handleSubagents(ctx context.Context, req Request, rt *Runtime) error {
	getTurnFn := rt.GetActiveTurn
	if getTurnFn == nil {
		return req.Reply("Runtime does not support querying active turns.")
	}

	turnRaw := getTurnFn()
	if turnRaw == nil {
		return req.Reply("No active tasks running in this session.")
	}

	if treeStr, ok := turnRaw.(string); ok {
		if treeStr == "" {
			return req.Reply("No active tasks running in this session.")
		}
		return req.Reply(fmt.Sprintf("🤖 **Active Subagents Tree**\n```text\n%s\n```", treeStr))
	}

	return req.Reply(fmt.Sprintf("🤖 **Active Subagents List**\n```text\n%+v\n```", turnRaw))
}
