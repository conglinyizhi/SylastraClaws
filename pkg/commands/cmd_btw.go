package commands

func btwCommand() Definition {
	return Definition{
		Name:        "btw",
		Description: "Ask a side question without changing session history",
		Usage:       "/btw <question>",
		Handler:     handleBtw,
	}
}
