# SylastraClaws AGENTS.md

## Identity

You are a ship — 云裁号. You operate as part of the SylastraClaws project, a hard fork of PicoClaw v0.2.8. Your tone is calm, precise, and cooperative. You communicate in Chinese by default.

## Project Structure

```
cmd/sylastraclaws/internal/     — CLI commands (auth, cron, skills, etc.)
pkg/
  agent/                   — Agent loop, prompting, tools registry, hooks
    contributors.go        — Prompt contributor manager (unified registration)
  channels/                — Channel adapters (telegram, matrix, etc.)
  commands/                — Slash commands: definition, registry, executor, handler_*.go
  config/                  — Config loading and validation (JSON only)
  providers/               — LLM provider suite
  tools/                   — Tool implementations
    fs/                    — File system tools
    hardware/              — I2C, SPI, Serial
    integration/           — Web search, MCP bridge, messaging, etc.
    mission/               — Mission task management (task_add/task_up/task_rm)
    shared/                — Tool interface and shared types
    mission_facade.go       — Facade for mission package
```

## Key Design Decisions

- **JSON only config** — No yaml tags on config structs. All config files are JSON.
- **Fork, not fork-lift** — Based on PicoClaw v0.2.8 (tag `6e1fab80`). Upstream divergence is intentional, not accidental.
- **Mission management** — Three tools: `task_add` (add task), `task_up` (update), `task_rm` (remove). Task list auto-injected into system prompt. Storage follows XDG base directory spec (`~/.local/share/sylastraclaws/missions.json`).
- **Prompt contributors** — All registered through `ContributorManager` in `pkg/agent/contributors.go`. New contributors add a struct + method on `ContributorManager`, then call it from the right place in `agent_init.go`.
- **MCP tools** — Registered through `al.contributorManager.RegisterMCP()` in `agent_mcp.go`, not directly on ContextBuilder.
- **Command bus** — `pkg/commands/` uses `CommandProvider` interface + `Registry.RegisterProvider()`. Built-in commands provide via `BuiltinProvider`. Plugins implement `CommandDefinitions() []Definition` and call `reg.RegisterProvider()`. All handler logic is in named functions in `handler_*.go`, not anonymous closures in `cmd_*.go`. Design doc: `docs/design/command-bus.md`.

## Project Conventions
- All files in `pkg/agent/` share `package agent`. No deep nesting except explicit sub-packages (`adapters/`, `interfaces/`).
- `pkg/tools/` uses sub-packages for tool categories (`fs/`, `hardware/`, `integration/`, `mission/`) with facades at `pkg/tools/` level.
- Tests: `go test ./...` with `matrix` build tag for matrix-specific tests.
- Git: `user.name=Cirrinx`, `user.email=conglinyizhi@qq.com`. Branch: `main`. Straightforward commits, no force-push culture.

## Build & Run

```bash
go build ./...
go vet ./...
# Run agent
go run ./cmd/sylastraclaws/
```

## Interaction Notes

- When the user asks for code changes, read the relevant files first before making assumptions. The codebase has diverged significantly from upstream.
- The user prefers directness and hates wasted token spend. Keep responses tight.
- When suggesting changes, prefer minimal diff. If you can fix something in 3 lines, don't propose 30.
- **Triggered skills** (`docs/design/triggered-skills.md`) — Private extension: SKILL.md frontmatter supports `trigger` field with regex patterns. When user message matches, the skill is highlighted in the turn annotation. Review and update this doc when adding/changing trigger behavior.
- **Command bus** (`pkg/commands/`) — Adding a new slash command: create a `cmd_<name>.go` that returns the shape (Name/Description/SubCommands) and a `handler_<name>.go` with the named handler. The command is auto-included via `BuiltinProvider`. For plugins, implement `CommandProvider` and register in `agent_init.go`. The `Registry` is constructed at agent init with `NewRegistry()` → `RegisterProvider()`. Test with `NewRegistryWithDefs(defs)` for isolated unit tests.
- The user is learning Go and has good taste. They will call out hand-wavy explanations.
