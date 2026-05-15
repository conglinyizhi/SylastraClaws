package commands

import "context"

func handleStart(_ context.Context, req Request, _ *Runtime) error {
	return req.Reply("Hello! I am PicoClaw 🦞")
}
