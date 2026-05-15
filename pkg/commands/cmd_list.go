package commands

func listCommand() Definition {
	return Definition{
		Name:        "list",
		Description: "List available options",
		SubCommands: []SubCommand{
			{
				Name:        "models",
				Description: "Configured models",
				Handler:     handleListModels,
			},
			{
				Name:        "channels",
				Description: "Enabled channels",
				Handler:     handleListChannels,
			},
			{
				Name:        "agents",
				Description: "Registered agents",
				Handler:     agentsHandler(),
			},
			{
				Name:        "skills",
				Description: "Installed skills",
				Handler:     handleListSkills,
			},
			{
				Name:        "mcp",
				Description: "Configured MCP servers",
				Handler:     listMCPServersHandler(),
			},
		},
	}
}
