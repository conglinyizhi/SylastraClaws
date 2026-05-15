package commands

// CommandProvider defines an interface for providing command definitions.
// Any package (including plugins) can implement this interface to contribute
// slash commands to the Registry.
type CommandProvider interface {
	CommandDefinitions() []Definition
}

// Registry stores the canonical command set used by both dispatch and
// optional platform registration adapters.
type Registry struct {
	defs  []Definition
	index map[string]int
}

// NewRegistry creates an empty Registry.
// Use RegisterProvider or the deprecated NewRegistryWithDefs to populate it.
func NewRegistry() *Registry {
	return &Registry{
		defs:  nil,
		index: make(map[string]int),
	}
}

// RegisterProvider registers all commands from the given provider.
// Duplicate command names are silently skipped (first registration wins).
func (r *Registry) RegisterProvider(provider CommandProvider) {
	defs := provider.CommandDefinitions()
	for _, def := range defs {
		if _, exists := r.index[normalizeCommandName(def.Name)]; exists {
			continue // first registration wins
		}
		idx := len(r.defs)
		r.defs = append(r.defs, def)
		registerCommandName(r.index, def.Name, idx)
		for _, alias := range def.Aliases {
			registerCommandName(r.index, alias, idx)
		}
	}
}

// NewRegistryWithDefs creates a Registry from a static definition list.
// Deprecated: use NewRegistry() + RegisterProvider() instead.
func NewRegistryWithDefs(defs []Definition) *Registry {
	r := NewRegistry()
	stored := make([]Definition, len(defs))
	copy(stored, defs)
	for _, def := range stored {
		idx := len(r.defs)
		r.defs = append(r.defs, def)
		registerCommandName(r.index, def.Name, idx)
		for _, alias := range def.Aliases {
			registerCommandName(r.index, alias, idx)
		}
	}
	return r
}

// Definitions returns all registered command definitions.
// Command availability is global and no longer channel-scoped.
func (r *Registry) Definitions() []Definition {
	out := make([]Definition, len(r.defs))
	copy(out, r.defs)
	return out
}

// Lookup returns a command definition by normalized command name or alias.
func (r *Registry) Lookup(name string) (Definition, bool) {
	key := normalizeCommandName(name)
	if key == "" {
		return Definition{}, false
	}
	idx, ok := r.index[key]
	if !ok {
		return Definition{}, false
	}
	return r.defs[idx], true
}

func registerCommandName(index map[string]int, name string, defIndex int) {
	key := normalizeCommandName(name)
	if key == "" {
		return
	}
	if _, exists := index[key]; exists {
		return
	}
	index[key] = defIndex
}
