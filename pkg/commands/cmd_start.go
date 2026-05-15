package commands

func startCommand() Definition {
	return Definition{
		Name:        "start",
		Description: "Start the bot",
		Usage:       "/start",
		Handler:     handleStart,
	}
}
