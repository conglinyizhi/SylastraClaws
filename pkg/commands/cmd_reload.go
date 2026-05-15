package commands

func reloadCommand() Definition {
	return Definition{
		Name:        "reload",
		Description: "Reload the configuration file",
		Usage:       "/reload",
		Handler:     handleReload,
	}
}
