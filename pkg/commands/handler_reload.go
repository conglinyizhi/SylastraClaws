package commands

import "context"

func handleReload(_ context.Context, req Request, rt *Runtime) error {
	if rt == nil || rt.ReloadConfig == nil {
		return req.Reply(unavailableMsg)
	}
	if err := rt.ReloadConfig(); err != nil {
		return req.Reply("Failed to reload configuration: " + err.Error())
	}
	return req.Reply("Config reload triggered!")
}
