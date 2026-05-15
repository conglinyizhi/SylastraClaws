# Triggered Skills (Private Extension)

> **Status**: Private extension — not part of agentskills.io or Hermes Agent standard.
> **Introduced**: May 2026, SylastraClaws fork.

## Purpose

Skills can declare regex patterns (triggers) in their SKILL.md frontmatter. When a user message matches any trigger, the skill name is highlighted in an ephemeral annotation appended to the current turn's user message. This is a purely additive hint — the LLM still decides whether to actually load and follow the skill.

Triggers solve a specific problem: in code-focused conversations, the LLM often overlooks relevant skills buried in the skill catalog. A short annotation right after the user's message draws attention without duplicating the full skill content into context.

## Frontmatter Format

```yaml
---
name: git-workflow
description: Git branch management, commit conventions, and PR workflow
trigger:
  - "git (push|commit|pull|rebase|merge|stash|reset)"
  - "PR|pull request"
  - "rebase|merge conflict"
---
```

- `trigger` is an array of Go regexp patterns.
- Each pattern is matched against the raw user message text.
- If any pattern matches, the skill name appears in the highlighted list.
- Invalid regex patterns are logged and silently skipped — they won't break loading.

## How It Works (per-turn)

1. User sends a message.
2. `ContextBuilder.BuildMessagesFromPrompt()` calls `skillsLoader.MatchTriggers(userMessage)`.
3. Matched skill names are collected (deduplicated, one match per skill is enough).
4. An annotation line is appended to the user message content:
   ```
   🔥 HIGHLIGHTED SKILLS — matched by built-in trigger rules: git-workflow, docker-deploy
   ```
5. This annotation is **not persisted** to conversation history — it's recomputed fresh every turn.

## Design Notes

- **No performance concern.** Go's `regexp.MatchString` on short user messages with ~50 patterns takes under 3ms.
- **No front-end matching.** Unlike agentskills.io's `platforms` field which is checked at the system level, triggers are purely a prompt-level hint. The engine doesn't load the skill — it just names it.
- **Not a replacement for active skills.** Active skills are explicitly requested (by name) and get their full content injected. Triggered skills only get a name mention.

## Example

A SKILL.md for Docker operations:

```yaml
---
name: docker-deploy
description: Docker Compose deployment, container management, and debugging
trigger:
  - "docker (compose|up|down|build|ps|logs|exec)"
  - "container.*(start|stop|restart|logs)"
  - "(dockerfile|Dockerfile)"
---
```

When the user types "docker compose up -d", the annotation `🔥 HIGHLIGHTED SKILLS — matched by built-in trigger rules: docker-deploy` is appended to the turn. The LLM sees this and can decide to read `SKILL.md` for `docker-deploy`.
