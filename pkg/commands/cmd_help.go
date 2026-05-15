package commands

func helpCommand() Definition {
	return Definition{
		Name:        "help",
		Description: "Show this help message",
		Usage:       "/help",
		Handler:     handleHelp,
	}
}
