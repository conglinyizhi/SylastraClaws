package commands

import "context"

func handleClear(_ context.Context, req Request, rt *Runtime) error {
	if rt == nil || rt.ClearHistory == nil {
		return req.Reply(unavailableMsg)
	}
	if err := rt.ClearHistory(); err != nil {
		return req.Reply("Failed to clear chat history: " + err.Error())
	}
	return req.Reply("Chat history cleared!")
}
