package commands

import (
	"context"
	"fmt"
	"strings"
)

func handleListModels(_ context.Context, req Request, rt *Runtime) error {
	if rt == nil || rt.GetModelInfo == nil {
		return req.Reply(unavailableMsg)
	}
	name, provider := rt.GetModelInfo()
	if provider == "" {
		provider = "configured default"
	}
	return req.Reply(fmt.Sprintf(
		"Configured Model: %s\nProvider: %s\n\nTo change models, update config.json",
		name, provider,
	))
}

func handleListChannels(_ context.Context, req Request, rt *Runtime) error {
	if rt == nil || rt.GetEnabledChannels == nil {
		return req.Reply(unavailableMsg)
	}
	enabled := rt.GetEnabledChannels()
	if len(enabled) == 0 {
		return req.Reply("No channels enabled")
	}
	return req.Reply(fmt.Sprintf("Enabled Channels:\n- %s", strings.Join(enabled, "\n- ")))
}

func handleListSkills(_ context.Context, req Request, rt *Runtime) error {
	if rt == nil || rt.ListSkillNames == nil {
		return req.Reply(unavailableMsg)
	}
	names := rt.ListSkillNames()
	if len(names) == 0 {
		return req.Reply("No installed skills")
	}
	return req.Reply(fmt.Sprintf(
		"Installed Skills:\n- %s\n\nUse /use <skill> <message> to force one for a single request, or /use <skill> to apply it to your next message.",
		strings.Join(names, "\n- "),
	))
}
