# Telegram 多话题群组（Forum Topics）适配方案

## 现状

目前已支持的 inbound/outbound 隔离：

- `telegram.go:handleMessage` → 构建 `compositeChatID = chatID/threadID`，仅在 `message.Chat.IsForum == true` 时
- `InboundContext.TopicID` 会设置
- `Bus.InboundContext` → `session.AllocateRouteSession` 通过 `shouldPreserveTelegramForumIsolation()` 将 topic 合入 chat 维度（当 policy 不含 `topic` 维度时）
- `sendChunk`、`SendMedia`、`StartTyping`、`SendPlaceholder`、`telegramStreamer` 均已传递 `MessageThreadID`
- `parseTelegramChatID` / `resolveTelegramOutboundTarget` 支持 `"chatID/threadID"` 格式

默认 session dimensions = `["chat"]`，因此每个 topic 天然有独立的 session key。

## 缺失功能

没有用户侧的**话题创建入口**。用户无法通过诸如 `/new` 等指令在管理群中创建新话题来获得独立上下文。

## 需求

当 bot 有群管理权限（`can_manage_topics`）时，用户可通过 `/new <话题标题>` 创建新话题。

Bot 收到 `/new` 指令后：
1. 判断当前 chat 是否为 forum 且 bot 是否有管理权限
2. 调用 Telegram `createForumTopic` API
3. 在新话题中发送一条欢迎/提示消息
4. 可选的：将 `chatID/threadID` 关联记录到某个存储，便于后续发现

## 设计方案

### 1. 实现方式选择

方案 A：作为**普通消息处理**的一部分（推荐）
- 用户在群中发 `/new 我的第二个项目`
- `handleMessage` 收到后检测到 content 以 `/new` 开头
- 在 `handleMessage` 中拦截，不走 agent 路由
- 调用 `createForumTopic`，在返回的 topic 中发一条消息
- 不影响现有 agent 流程

方案 B：作为**斜杠指令**注册
- 利用现有的 `commands` 系统注册 `/new` 指令
- 但需要 channel 层能调用 Telegram 专属 API（`CreateForumTopic`）
- 需要新增 `ForumCapable` 接口或直接在 TelegramChannel 上暴露方法

采用方案 A 的理由：`/new` 是 Telegram 平台特有能力，放在 Telegram channel 实现中比抽象成通用 commands 更自然。commands 系统的 handler 与 channel 实例没有直接关联，要调用 `bot.CreateForumTopic` 需要额外 wire。

### 2. 指令格式

```
/new <话题标题>
/新的 <话题标题>
```

别名支持中英文。

### 3. 前置检查

```go
if message.Chat.Type != "supergroup" || !message.Chat.IsForum {
    // 回复 "此群组未启用话题功能"
    return nil
}
```

### 4. 流程

1. `handleMessage` 中，在 allowlist 检查后、`IsForum` compositeChatID 构建之前，拦截 `/new` 指令
2. 调用 `bot.CreateForumTopic(ctx, &telego.CreateForumTopicParams{ChatID: tu.ID(chatID), Name: title})`
3. 成功 → 在新 topic 中发送一条消息
4. 失败 → 在原消息回复错误信息

### 5. 实现位置

`pkg/channels/telegram/telegram.go`，新增方法：

```go
func (c *TelegramChannel) handleNewTopic(ctx context.Context, message *telego.Message) (bool, error)
```

返回 `(handled bool, err error)`，`handled=true` 表示已处理，`handleMessage` 跳过后续流程。

### 6. 权限检查

Telegram Bot API 没有直接查询 bot 自身在群中权限的 endpoint。使用**乐观方式**：直接调用 `CreateForumTopic`，失败时提示用户确认权限。

### 7. 后续可扩展

- `/list` 列出所有活跃话题及其 session 状态
- `/close` 关闭当前话题（`closeForumTopic` / `deleteForumTopic`）
- `/archive` 归档话题

## 不涉及改动

- session 分配逻辑：已有 topic 维度隔离，无需改变
- routing 逻辑：`/new` 是 channel 层拦截，不进 agent
- `commands` 系统：不增加 `ForumTopicCommandProvider`
- 配置：无需新增配置项

## 实现规模

- 新增 1 个 handler 方法：~50 行
- 修改 `handleMessage` 中约 3 行的条件判断
- 总改动：~60 行，在 `telegram.go` 内完成
