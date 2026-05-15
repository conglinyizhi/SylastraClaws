package commands

func clearCommand() Definition {
	return Definition{
		Name:        "clear",
		Description: "Clear the chat history",
		Usage:       "/clear",
		Handler:     handleClear,
	}
}
