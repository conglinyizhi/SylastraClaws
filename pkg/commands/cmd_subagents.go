package commands

// TurnInfo is a mirrored struct from agent.TurnInfo to avoid circular dependencies.
type TurnInfo struct {
	TurnID       string
	ParentTurnID string
	Depth        int
	ChildTurnIDs []string
	IsFinished   bool
}

func subagentsCommand() Definition {
	return Definition{
		Name:        "subagents",
		Description: "Show running subagents and task tree",
		Handler:     handleSubagents,
	}
}
