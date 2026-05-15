<div align="center">

<h1>SylastraClaws: PicoClaw Reforged</h1>

<h3>Go AI Assistant — Fork of PicoClaw v0.2.8</h3>
  <p>
    <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go&logoColor=white" alt="Go">
    <img src="https://img.shields.io/badge/license-MIT-green" alt="License">
  </p>

[中文](docs/project/README.zh.md) | **English**

</div>

> ⚠️ **SylastraClaws** is a hard fork of [PicoClaw](https://github.com/sipeed/picoclaw) at tag **v0.2.8**, forked for deep customization and architectural divergence. This is not an official PicoClaw derivative or distribution.

---

## Name & Pronunciation

**Sylastra** /sɪˈlæstrə/ or /saɪˈlæstrə/

- *Syl-* from Latin *silva / sylva* — forest
- *-astra* from Latin *astra* — stars, constellations

Together: **Forest of Stars**, or **The Starwood**.

**Claws** — like *OpenClaw*, the name carries the same analogy: a claw that reaches out, grabs, and acts. Not a passive tool — active, capable, autonomous.

> **Sylastra Claws** — The Claw of the Starwood.
> 
> *Sylastra* is the carrier, the mothership from which all sub-agents are born. *Claws* are the reaching hands — the extensions that give LLMs the physical ability to grip, execute, and act in the system.

### What This Fork Is

This project took PicoClaw's ultra-lightweight AI agent infrastructure and redirected it toward a different design philosophy — replacing the original file-system tools with [better-edit-tools-mcp](https://github.com/conglinyizhi/better-edit-tools-mcp) (v0.6.0) as the native read/write filesystem layer, with ongoing divergence in both tooling and architecture.

### Why fork instead of contributing upstream?

The scope of changes — deep tool replacement, architectural restructuring, and opinionated choices about LLM interaction patterns — is incompatible with PicoClaw's original design direction. A fork is the honest path.

### What Has Been Done Since the Fork

| Category | Changes |
|---|---|
| **File System** | Replaced in-tree file tools with betools v0.6.0 (better-read/write/insert/delete/patch/batch suite). Added binary detection, memory:// protocol path resolution, be-trx-rollback for edit transaction rollback. |
| **Config** | Switched to JSON-only config (removed TOML dual support). All config files are JSON; YAML tags removed from config structs. |
| **Frontend** | Entire web/ directory removed (-56K lines). Dashboard, frontend, associated npm dependencies cut. |
| **CI** | All GitHub Actions switched to `workflow_dispatch` only (manual trigger). Dependabot PRs (#6-#9) merged. |
| **Matrix Channel** | Gated behind `matrix` build tag — excluded from default builds (no CGO/libolm dependency). |
| **Tool Architecture** | Renamed internal tools per `better_*` convention via shared facade aliases. `ExecTool` now supports process tree inspection (`list-tree` action). |
| **Channel Integration** | Telegram, Feishu, WeChat, QQBot active. Matrix opt-in. |
| **Licensing** | MIT — upstream copyright preserved, modifications carry own copyright. |
| **Mission Management** | Built-in mission/task system (task_add/task_up/task_rm tools). Tasks auto-injected into system prompt. |
| **Skill Triggers** | Private extension: SKILL.md frontmatter supports `trigger` field with regex. Matching skills highlighted in each turn (ephemeral). |
| **Prompt Architecture** | Unified prompt contributor registry, multi-layer prompt stack with caching, flat map injector for AGENTS.md. |

### Key Technical Differences from Upstream

| Area | PicoClaw | SylastraClaws |
|---|---|---|
| **File System Tools** | Custom in-tree implementations | betools v0.6.0 (better-read/write suite + trx-rollback) |
| **Go Version** | 1.25+ | 1.26+ |
| **Tool Architecture** | Flat tool set | Namespaced `better_*` convention |
| **Binary Detection** | None in read path | Built into betools Read() |
| **Injectable FS** | Static path validation | betools FileSystem interface ready |
| **Batch Editing** | Per-file operations | betools Batch / WriteFilesAtomic |
| **Focus** | Embedded/$10 hardware | Desktop/cloud agent infrastructure |

| **Task Management** | None | task_add / task_up / task_rm tools, auto-injected into prompt |
| **Skill Matching** | None | Trigger regex patterns in SKILL.md frontmatter |
| **Config Format** | YAML+TOML dual | JSON-only |

### Design Documentation

For technical details on architecture decisions and extensions, see:
- [Hook System](docs/architecture/hooks/README.md)
- [Mission System](docs/design/mission-management.md)
- [Triggered Skills](docs/design/triggered-skills.md) — private extension for regex-based skill highlighting
- [Prompt Architecture](docs/design/prompt-injection.md) — prompt contributor registry & flat map injector

### License

MIT — see [LICENSE](LICENSE). This project retains the copyright of PicoClaw contributors (the original work) and adds copyright for modifications. betools (v0.6.0) is also MIT-licensed.
