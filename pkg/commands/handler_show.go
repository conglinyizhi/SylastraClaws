package commands

import (
	"context"
	"fmt"
)

func handleShowModel(_ context.Context, req Request, rt *Runtime) error {
	if rt == nil || rt.GetModelInfo == nil {
		return req.Reply(unavailableMsg)
	}
	name, provider := rt.GetModelInfo()
	return req.Reply(fmt.Sprintf("Current Model: %s (Provider: %s)", name, provider))
}

func handleShowChannel(_ context.Context, req Request, _ *Runtime) error {
	return req.Reply(fmt.Sprintf("Current Channel: %s", req.Channel))
}
