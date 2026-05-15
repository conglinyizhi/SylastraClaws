package commands

import (
	"context"
	"fmt"
	"strings"
)

func handleHelp(_ context.Context, req Request, rt *Runtime) error {
	var defs []Definition
	if rt != nil && rt.ListDefinitions != nil {
		defs = rt.ListDefinitions()
	} else {
		defs = BuiltinProvider{}.CommandDefinitions()
	}
	return req.Reply(formatHelpMessage(defs))
}

func formatHelpMessage(defs []Definition) string {
	if len(defs) == 0 {
		return "No commands available."
	}

	lines := make([]string, 0, len(defs))
	for _, def := range defs {
		usage := def.EffectiveUsage()
		if usage == "" {
			usage = "/" + def.Name
		}
		desc := def.Description
		if desc == "" {
			desc = "No description"
		}
		lines = append(lines, fmt.Sprintf("%s - %s", usage, desc))
	}
	return strings.Join(lines, "\n")
}
