<div align="center">

<h1>SylastraClaws: PicoClaw Reforged</h1>

<h3>Go AI Assistant — Fork of PicoClaw v0.2.8</h3>
  <p>
    <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go&logoColor=white" alt="Go">
    <img src="https://img.shields.io/badge/license-MIT-green" alt="License">
  </p>

[中文](docs/project/README.zh.md) | **English**

> ⚠️ **SylastraClaws** is a hard fork of [PicoClaw](https://github.com/sipeed/picoclaw) at tag **v0.2.8**, forked for deep customization and architectural divergence. This is not an official PicoClaw derivative or distribution.

**What is this?**

SylastraClaws takes PicoClaw's ultra-lightweight AI agent infrastructure and redirects it toward a different design philosophy — replacing the original file-system tools with [better-edit-tools-mcp](https://github.com/conglinyizhi/better-edit-tools-mcp) (v0.5.0) as the native read/write filesystem layer, with ongoing divergence in both tooling and architecture.

**Why fork instead of contributing upstream?**

The scope of changes — deep tool replacement, architectural restructuring, and opinionated choices about LLM interaction patterns — is incompatible with PicoClaw's original design direction. A fork is the honest path.

### Key Differences from PicoClaw

| Area | PicoClaw | SylastraClaws |
|---|---|---|
| **File System Tools** | Custom in-tree implementations | betools v0.5.0 (better-read/write suite) |
| **Go Version** | 1.25+ | 1.22+ (widened, same features) |
| **Tool Architecture** | Flat tool set | Namespaced `better_*` convention |
| **Binary Detection** | None in read path | Built into betools Read() |
| **Injectable FS** | Static path validation | betools FileSystem interface ready |
| **Batch Editing** | Per-file operations | betools Batch / WriteFilesAtomic |
| **Focus** | Embedded/$10 hardware | Desktop/cloud agent infrastructure |

### License

MIT — see [LICENSE](LICENSE). This project retains the copyright of PicoClaw contributors (the original work) and adds copyright for modifications. betools (v0.5.0) is also MIT-licensed.
