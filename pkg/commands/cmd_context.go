package commands

func contextCommand() Definition {
	return Definition{
		Name:        "context",
		Description: "Show current session context and token usage",
		Usage:       "/context",
		Handler:     handleContext,
	}
}
