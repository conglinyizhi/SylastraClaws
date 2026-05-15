# Mission Management System

## Overview

The mission system provides built-in task/mission tracking for SylastraClaws agents. Three tools (`task_add`, `task_up`, `task_rm`) manage a JSON-backed task store, and the current task list is automatically injected into the system prompt every turn.

This is a private extension — not part of upstream PicoClaw.

## Data Storage

- **File**: `$XDG_DATA_HOME/sylastraclaws/missions.json`
- **Format**: JSON with `next_id` counter and `items` array
- **XDG fallback**: `~/.local/share/sylastraclaws/missions.json` when `$XDG_DATA_HOME` is unset
- **Atomic writes**: Uses `.tmp` → `rename` pattern to avoid corruption

### JSON Schema

```json
{
  "next_id": 5,
  "items": [
    {
      "id": 1,
      "title": "Set up CI pipeline",
      "description": "Add GitHub Actions workflow",
      "status": "done",
      "priority": 1,
      "created_at": "2026-05-13T12:00:00Z",
      "updated_at": "2026-05-13T14:30:00Z"
    }
  ]
}
```

## Tools

### `task_add` — Add a new task

| Parameter     | Type   | Required | Description                         |
|---------------|--------|----------|-------------------------------------|
| `title`       | string | yes      | Short task title                    |
| `description` | string | no       | Task notes or details               |
| `priority`    | int    | no       | 1=high, 2=medium (default), 3=low   |
| `status`      | string | no       | `pending` (default) or `done`       |

Returns: `[pending] <title> (id:<N> pri:<N>)`

### `task_up` — Update a task

| Parameter     | Type   | Required | Description                         |
|---------------|--------|----------|-------------------------------------|
| `id`          | int    | yes      | Task ID                             |
| `title`       | string | no       | New title                           |
| `description` | string | no       | New description                     |
| `status`      | string | no       | `pending`, `done`, or `closed`      |
| `priority`    | int    | no       | 1=high, 2=medium, 3=low             |

Returns: `updated #<N> (<changed-fields>): [<status>] <title> (id:<N> pri:<N>)`

### `task_rm` — Remove a task

| Parameter | Type | Required | Description           |
|-----------|------|----------|-----------------------|
| `id`      | int  | yes      | Task ID to remove     |

Returns: `[deleted] [<status>] <title> (id:<N> pri:<N>)`

## Prompt Injection

Every turn, the `ContextBuilder.buildDynamicContext()` method reads the current mission list and injects it as:

```
## Active Missions
- [pending] Set up CI pipeline (id:1, pri:1)
- [done] Deploy v1.0 (id:2, pri:1)
- [pending] Write documentation (id:3, pri:2)
```

This is part of the dynamic context (time, runtime, session info), not cached in the system prompt. The list refreshes every turn from the JSON file, so any `task_add`/`task_up`/`task_rm` call from a previous turn is immediately reflected.

## Implementation

### Files

- `pkg/tools/mission/types.go` — `MissionItem` struct
- `pkg/tools/mission/store.go` — `Store` (thread-safe JSON persistence with `sync.RWMutex`)
- `pkg/tools/mission/tools.go` — `AddTool`, `UpdateTool`, `RemoveTool`
- `pkg/tools/mission_facade.go` — `NewMissionTools()` constructor for agent registration
- `pkg/agent/agent_init.go` — Registration: calls `tools.NewMissionTools()` when `mission` tool is enabled
- `pkg/agent/context.go` — Injection: `buildDynamicContext()` reads `missionStore.List()` and formats tasks

### Safety

- All mutations (Add/Update/Remove) use `sync.RWMutex` for thread safety
- File writes are atomic (tmp → rename)
- Store initialization failure is non-fatal — `initMissionStore()` returns `nil` on error, and `buildDynamicContext()` silently skips injection when store is nil
