package commands

func thinkCommand() Definition {
	return Definition{
		Name:        "think",
		Description: "Change thinking level",
		SubCommands: []SubCommand{
			{
				Name:        "to",
				Description: "Set thinking level (off|low|medium|high|xhigh|adaptive)",
				ArgsUsage:   "<level>",
				Handler:     handleSwitchThinkLevel,
			},
		},
	}
}
