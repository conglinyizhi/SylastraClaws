package commands

func showCommand() Definition {
	return Definition{
		Name:        "show",
		Description: "Show current configuration",
		SubCommands: []SubCommand{
			{
				Name:        "model",
				Description: "Current model and provider",
				Handler:     handleShowModel,
			},
			{
				Name:        "channel",
				Description: "Current channel",
				Handler:     handleShowChannel,
			},
			{
				Name:        "agents",
				Description: "Registered agents",
				Handler:     agentsHandler(),
			},
			{
				Name:        "mcp",
				Description: "Active tools for an MCP server",
				ArgsUsage:   "<server>",
				Handler:     showMCPToolsHandler(),
			},
		},
	}
}
