package commands

// BuiltinProvider provides all built-in command definitions.
// Each command group is defined in its own cmd_*.go file.
type BuiltinProvider struct{}

func (BuiltinProvider) CommandDefinitions() []Definition {
	return []Definition{
		startCommand(),
		helpCommand(),
		showCommand(),
		listCommand(),
		useCommand(),
		btwCommand(),
		switchCommand(),
		checkCommand(),
		clearCommand(),
		contextCommand(),
		subagentsCommand(),
		reloadCommand(),
	}
}
