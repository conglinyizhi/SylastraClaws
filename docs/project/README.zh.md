<div align="center">

<h1>SylastraClaws: 重铸 PicoClaw</h1>

<h3>Go AI 助手 — Fork 自 PicoClaw v0.2.8</h3>
  <p>
    <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go&logoColor=white" alt="Go">
    <img src="https://img.shields.io/badge/license-MIT-green" alt="License">
  </p>

**中文** | [English](../../README.md)

</div>

---

> ⚠️ **SylastraClaws** 是 [PicoClaw](https://github.com/sipeed/picoclaw) 的硬分叉，fork 自标签 **v0.2.8**。本项目的目标与上游不同，将在架构和工具集上持续分化。这不是 PicoClaw 的官方衍生版本或发行版。

---

## 名字由来与读音

**Sylastra** /sɪˈlæstrə/ 或 /saɪˈlæstrə/

- *Syl-* 来自拉丁语 *silva / sylva* — 森林
- *-astra* 来自拉丁语 *astra* — 群星、星辰之群

合起来：**星群之森**，或意译为 **星辰之森**。

**Claws（利爪）**——和 OpenClaw 的命名逻辑一样：一个能主动伸出去抓取、接手的爪子。不是被动的工具，而是能自主行动的 AI 执行体。

> **Sylastra Claws** —— 星群之森的利爪，或称星辰之爪。
>
> *Sylastra* 是载体，是母舰，是所有子 Agent 诞生的调度平台。*Claws* 是延伸出去的执行之手——赋予 LLM 真正在系统里"上手做事"的物理化能力。

---

## 这是什么？

SylastraClaws 继承了 PicoClaw 的超轻量 AI Agent 基础设施，但将其导向了不同的设计方向——用 [better-edit-tools-mcp](https://github.com/conglinyizhi/better-edit-tools-mcp)（v0.5.0）替换了原生的文件系统工具层，并在工具架构和交互模式上持续分化。

## 为什么 fork 而不是向上游贡献？

变更范围——深层的工具替换、架构重组、以及对 LLM 交互模式的主观取舍——与 PicoClaw 的设计方向不再兼容。fork 是更诚实的路径。

## Fork 后的主要改动

| 类别 | 改动 |
|---|---|
| **文件系统** | 用 betools v0.5.0（better-read/write/insert/delete/patch/batch 套件）替换内建工具。新增二进制检测、memory:// 协议路径解析。 |
| **配置** | 从纯 JSON 改为 TOML + JSON 双支持（LoadConfig 自动识别后缀）。.golangci.yaml、.goreleaser.yaml 迁移为 TOML。 |
| **前端** | 移除整个 web/ 目录（-56K 行），仪表盘、前端、相关 npm 依赖全部删除。 |
| **CI** | 所有 GitHub Actions 改为仅 `workflow_dispatch` 手动触发。已合并 Dependabot PR（#6-#9）。 |
| **Matrix 通道** | 通过 `matrix` build tag 隔离——默认构建不包含（无需 CGO/libolm 依赖）。 |
| **工具架构** | 按 `better_*` 命名约定重命名工具（共用 facade 别名）。ExecTool 新增进程树查看（`list-tree` action）。 |
| **通道集成** | Telegram / Feishu / WeChat / QQBot 活跃集成。Matrix 为 opt-in。 |
| **开源许可** | MIT——上游版权保留，修改部分独立版权。 |

## 与 PicoClaw 的主要技术差异

| 方面 | PicoClaw | SylastraClaws |
|---|---|---|
| **文件系统工具** | 内建实现 | betools v0.5.0（better-read/write 套件） |
| **Go 版本** | 1.25+ | 1.26+ |
| **工具架构** | 平铺式工具集 | `better_*` 命名空间约定 |
| **二进制检测** | 读路径无检测 | 内置在 betools Read() 中 |
| **可注入文件系统** | 静态路径校验 | betools FileSystem 接口 |
| **批量编辑** | 逐文件操作 | betools Batch / WriteFilesAtomic |
| **关注点** | 嵌入式 / $10 硬件 | 桌面 / 云端 Agent 基础设施 |

## 开源许可

MIT — 见 [LICENSE](../../LICENSE)。本项目保留了 PicoClaw 贡献者的版权（原始作品），并新增了修改部分的版权。betools (v0.5.0) 同样使用 MIT 协议。
