package commands

import (
	"context"
	"fmt"
)

func handleSwitchModel(_ context.Context, req Request, rt *Runtime) error {
	if rt == nil || rt.SwitchModel == nil {
		return req.Reply(unavailableMsg)
	}
	// Parse: /switch model to <value>
	value := nthToken(req.Text, 3) // tokens: [/switch, model, to, <value>]
	if nthToken(req.Text, 2) != "to" || value == "" {
		return req.Reply("Usage: /switch model to <name>")
	}
	oldModel, err := rt.SwitchModel(value)
	if err != nil {
		return req.Reply(err.Error())
	}
	return req.Reply(fmt.Sprintf("Switched model from %s to %s", oldModel, value))
}
