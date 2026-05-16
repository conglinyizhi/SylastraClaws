package commands

import (
	"context"
	"fmt"
	"strings"
)

var validThinkLevels = []string{"off", "low", "medium", "high", "xhigh", "adaptive"}

func handleSwitchThinkLevel(_ context.Context, req Request, rt *Runtime) error {
	if rt == nil || rt.SwitchThinkLevel == nil {
		return req.Reply(unavailableMsg)
	}
	// Parse: /think to <level>
	if nthToken(req.Text, 1) != "to" {
		return req.Reply(thinkUsage())
	}
	level := strings.ToLower(strings.TrimSpace(nthToken(req.Text, 2)))
	if level == "" {
		return req.Reply(thinkUsage())
	}
	if !isValidThinkLevel(level) {
		valid := strings.Join(validThinkLevels, ", ")
		return req.Reply(fmt.Sprintf("Invalid thinking level %q. Valid levels: %s", level, valid))
	}
	oldLevel, err := rt.SwitchThinkLevel(level)
	if err != nil {
		return req.Reply(err.Error())
	}
	return req.Reply(fmt.Sprintf("Switched thinking from %s to %s", oldLevel, level))
}

func isValidThinkLevel(level string) bool {
	for _, v := range validThinkLevels {
		if level == v {
			return true
		}
	}
	return false
}

func thinkUsage() string {
	valid := strings.Join(validThinkLevels, ", ")
	return fmt.Sprintf("Usage: /think to <%s>", valid)
}
