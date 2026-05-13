<div align="center">
<img src="../../assets/logo.webp" alt="SylastraClaws" width="512">

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

**这是什么？**

SylastraClaws 继承了 PicoClaw 的超轻量 AI Agent 基础设施，但将其导向了不同的设计方向——用 [better-edit-tools-mcp](https://github.com/conglinyizhi/better-edit-tools-mcp)（v0.5.0）替换了原生的文件系统工具层，并在工具架构和交互模式上持续分化。

**为什么 fork 而不是向上游贡献？**

变更范围——深层的工具替换、架构重组、以及对 LLM 交互模式的主观取舍——与 PicoClaw 的设计方向不再兼容。fork 是更诚实的路径。

### 与 PicoClaw 的主要差异

**文件系统工具** — 上游为内建实现，SylastraClaws 使用 betools v0.5.0（better-read/write 套件）

**Go 版本** — 上游要求 1.25+，SylastraClaws 放宽到 1.22+（特性一致）

**工具架构** — 上游为平铺式工具集，SylastraClaws 采用 `better_*` 命名空间约定

**二进制文件检测** — 上游读路径无检测，SylastraClaws 内置在 betools Read() 中

**可注入文件系统** — 上游使用静态路径校验，SylastraClaws 支持 betools FileSystem 接口

**批量编辑** — 上游逐文件操作，SylastraClaws 支持 betools Batch / WriteFilesAtomic

**关注点** — 上游面向嵌入式/$10 硬件，SylastraClaws 面向桌面/云端 Agent 基础设施

### 开源许可

MIT — 见 [LICENSE](../../LICENSE)。本项目保留了 PicoClaw 贡献者的版权（原始作品），并新增了修改部分的版权。betools (v0.5.0) 同样使用 MIT 协议。
