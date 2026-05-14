package tools

import (
	"github.com/conglinyizhi/SylastraClaws/pkg/tools/mission"
)

// NewMissionTools creates the three mission task management tools backed
// by an XDG-compliant JSON store. Caller must check IsToolEnabled("mission")
// before calling this.
func NewMissionTools() ([]Tool, error) {
	store, err := mission.NewStore()
	if err != nil {
		return nil, err
	}
	return []Tool{
		mission.NewAddTool(store),
		mission.NewUpdateTool(store),
		mission.NewRemoveTool(store),
	}, nil
}
