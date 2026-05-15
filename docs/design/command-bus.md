# Command Bus

## 动机

原始的命令系统有两个问题：

1. **匿名函数堆积** — 每个 `cmd_*.go` 把 handler 实现直接写在 `SubCommand.Handler` 字段里。无法单独测试，容易重复，`/show agents` 和 `/list agents` 做了类似的事但各写一遍。
2. **固定注册表** — `BuiltinDefinitions()` 静态返回所有命令，外部包 / 未来插件无法注入新命令。Registry 构建后不可变。

## 改动

### 1. 命名 Handler

所有 `SubCommand` 中原本的匿名函数被提取为包级命名函数，放在 `handler_*.go` 中。`cmd_*.go` 仅负责定义命令形状（Name/Description/Aliases/SubCommands），不再包含业务逻辑。

命名 handler 的命名规则：`handle<Command><SubCommand>`，如 `handleShowModel`, `handleListModels`。

可复用的 handler（如 agents 列表）导出为公共函数。

### 2. CommandProvider 接口

```go
type CommandProvider interface {
    CommandDefinitions() []Definition
}
```

任何包实现此接口即可注册自己的命令。

### 3. Registry 可变

`Registry` 新增 `RegisterProvider(provider CommandProvider)` 方法，在初始化阶段多次调用来聚合所有命令：

```go
reg := commands.NewRegistry()
reg.RegisterProvider(commands.NewBuiltinProvider())  // 内置命令
reg.RegisterProvider(myplugin.NewCommandProvider())   // 插件命令
```

`NewRegistry()` 不再接收 `[]Definition` 参数，改为无参构造。`BuiltinDefinitions()` 被 `BuiltinProvider` 替代。

### 4. `/switch channel` 清理

废弃的 `/switch channel` 子命令定义被移除。`/check channel` 成为检查通道可用性的唯一入口。

## 文件变化

| 文件 | 变化 |
|------|------|
| `registry.go` | `NewRegistry()` 无参；新增 `RegisterProvider()` |
| `builtin.go` | 替换为 `BuiltinProvider` |
| `cmd_*.go` | 去掉 handler 实现，只保留 Definition 形状 |
| `handler_*.go` | 新增/重组：所有命名 handler 集中于此 |
| `handler_agents.go` | agents handler（复用 /show 和 /list） |
| `handler_mcp.go` | MCP handler（/show mcp, /list mcp） |
| `executor.go` | 无需改动（已基于 Registry 工作） |
| `agent_command.go` | 使用新的 `NewRegistry()` 无参构造 |
