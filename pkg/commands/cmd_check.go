package commands

func checkCommand() Definition {
	return Definition{
		Name:        "check",
		Description: "Check channel availability",
		SubCommands: []SubCommand{
			{
				Name:        "channel",
				Description: "Check if a channel is available",
				ArgsUsage:   "<name>",
				Handler:     handleCheckChannel,
			},
		},
	}
}
