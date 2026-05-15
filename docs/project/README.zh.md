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

## 快速开始

### 首次设置

```bash
# 编译并一键配置
go build -o sylastraclaws ./cmd/picoclaw/

# 交互式首次配置 —— API key、模型名、中转地址任意顺序
./sylastraclaws --first-run "sk-xxx,gpt-4o"
./sylastraclaws --first-run "sk-ant-xxx,claude-sonnet-4-20250514,https://api.anthropic.com"

# 开始聊天
./sylastraclaws agent
```

`--first-run` 自动识别提供商（OpenAI、Anthropic、Gemini、DeepSeek 等 20+），发送测试消息验证，然后写入配置文件。无需手动编辑。

### 全量编译（含 Matrix 通道）

```bash
# 日常开发 —— 不含 Matrix（无 CGo 依赖）
go build ./...

# 全量编译 —— 含 Matrix 通道（纯 Go 加密，零 CGo）
make build-all
```

Matrix 通道需要同时使用 `matrix` 和 `goolm` 两个 build tag（`make build-all` 会自动处理）。默认 `go build ./...` 完全排除 Matrix。

---

## 这是什么？

SylastraClaws 继承了 PicoClaw 的超轻量 AI Agent 基础设施，但将其导向了不同的设计方向——用 [better-edit-tools-mcp](https://github.com/conglinyizhi/better-edit-tools-mcp)（v0.6.0）替换了原生的文件系统工具层，并在工具架构和交互模式上持续分化。

## 为什么 fork 而不是向上游贡献？

变更范围——深层的工具替换、架构重组、以及对 LLM 交互模式的主观取舍——与 PicoClaw 的设计方向不再兼容。fork 是更诚实的路径。

## Fork 后的主要改动

| 类别 | 改动 |
|---|---|
| **文件系统** | 用 betools v0.6.0（better-read/write/insert/delete/patch/batch 套件 + be-trx-rollback 编辑事务回滚）替换内建工具。新增二进制检测、memory:// 协议路径解析。 |
| **配置** | 改为纯 JSON 配置（移除 TOML 双支持）。所有配置文件使用 JSON；config struct 移除 YAML 标签。 |
| **前端** | 移除整个 web/ 目录（-56K 行），仪表盘、前端、相关 npm 依赖全部删除。 |
| **CI** | 所有 GitHub Actions 改为仅 `workflow_dispatch` 手动触发。已合并 Dependabot PR（#6-#9）。 |
| **Matrix 通道** | 通过 `matrix` build tag 隔离——默认构建不包含（无需 CGO/libolm 依赖）。 |
| **工具架构** | 按 `better_*` 命名约定重命名工具（共用 facade 别名）。ExecTool 新增进程树查看（`list-tree` action）。 |
| **通道集成** | Telegram / Feishu / WeChat / QQBot 活跃集成。Matrix 为 opt-in。 |
| **开源许可** | MIT——上游版权保留，修改部分独立版权。 |
| **使命管理** | 内置使命/任务系统（task_add/task_up/task_rm 工具）。任务列表自动注入 system prompt。 |
| **技能触发** | 私有扩展：SKILL.md frontmatter 支持 `trigger` 正则字段。匹配的技能在每一轮短暂高亮（不写入历史）。 |
| **提示词架构** | 统一 Prompt Contributor 注册机制，多层 prompt stack 缓存，AGENTS.md 扁平注入引擎。 |

## 与 PicoClaw 的主要技术差异

| 方面 | PicoClaw | SylastraClaws |
|---|---|---|
| **文件系统工具** | 内建实现 | betools v0.6.0（better-read/write 套件 + trx-rollback） |
| **Go 版本** | 1.25+ | 1.26+ |
| **工具架构** | 平铺式工具集 | `better_*` 命名空间约定 |
| **二进制检测** | 读路径无检测 | 内置在 betools Read() 中 |
| **可注入文件系统** | 静态路径校验 | betools FileSystem 接口 |
| **批量编辑** | 逐文件操作 | betools Batch / WriteFilesAtomic |
| **关注点** | 嵌入式 / $10 硬件 | 桌面 / 云端 Agent 基础设施 |

| **任务管理** | 无 | task_add / task_up / task_rm 工具，自动注入 prompt |
| **技能匹配** | 无 | SKILL.md frontmatter 中的 trigger 正则匹配 |
| **配置格式** | YAML+TOML 双支持 | 纯 JSON |

### 设计文档

技术架构决策与扩展的详细说明：
- [钩子系统](docs/architecture/hooks/README.zh.md)
- [使命系统](docs/design/mission-management.md)
- [技能触发高亮](docs/design/triggered-skills.md) — 基于正则的技能高亮私有扩展
- [提示词注入架构](docs/design/prompt-injection.md) — Prompt Contributor 注册机制 & 扁平注入引擎

## 开源许可

MIT — 见 [LICENSE](../../LICENSE)。本项目保留了 PicoClaw 贡献者的版权（原始作品），并新增了修改部分的版权。betools (v0.5.0) 同样使用 MIT 协议。
