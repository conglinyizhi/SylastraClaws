package commands

func switchCommand() Definition {
	return Definition{
		Name:        "switch",
		Description: "Switch model",
		SubCommands: []SubCommand{
			{
				Name:        "model",
				Description: "Switch to a different model",
				ArgsUsage:   "to <name>",
				Handler:     handleSwitchModel,
			},
		},
	}
}
