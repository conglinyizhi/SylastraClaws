// SylastraClaws - Ultra-lightweight personal AI agent

package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/conglinyizhi/SylastraClaws/pkg/constants"
	"github.com/conglinyizhi/SylastraClaws/pkg/logger"
	"github.com/conglinyizhi/SylastraClaws/pkg/providers"
)

// CallLLM performs an LLM call with fallback support, hook invocation, and retry logic.
// It handles PreLLM setup, the actual LLM invocation with retry, and AfterLLM processing.
// Returns Control indicating what the coordinator should do next.
func (p *Pipeline) CallLLM(
	ctx context.Context,
	turnCtx context.Context,
	ts *turnState,
	exec *turnExecution,
	iteration int,
) (Control, error) {
	al := p.al
	maxMediaSize := p.Cfg.Agents.Defaults.GetMaxMediaSize()

	// PreLLM: resolve media refs (except on iteration 1 where user media is already resolved)
	if iteration > 1 {
		exec.messages = resolveMediaRefs(exec.messages, p.MediaStore, maxMediaSize)
	}

	// PreLLM: graceful terminal handling
	exec.gracefulTerminal, _ = ts.gracefulInterruptRequested()
	exec.providerToolDefs = ts.agent.Tools.ToProviderDefs()

	// Native web search support
	webSearchEnabled := al.cfg.Tools.IsToolEnabled("web")
	exec.useNativeSearch = webSearchEnabled && al.cfg.Tools.Web.PreferNative &&
		func() bool {
			if ns, ok := ts.agent.Provider.(providers.NativeSearchCapable); ok {
				return ns.SupportsNativeSearch()
			}
			return false
		}()

	if exec.useNativeSearch {
		filtered := make([]providers.ToolDefinition, 0, len(exec.providerToolDefs))
		for _, td := range exec.providerToolDefs {
			if td.Function.Name != "web_search" {
				filtered = append(filtered, td)
			}
		}
		exec.providerToolDefs = filtered
	}

	exec.callMessages = exec.messages
	if exec.gracefulTerminal {
		exec.callMessages = append(append([]providers.Message(nil), exec.messages...), ts.interruptHintMessage())
		exec.providerToolDefs = nil
		ts.markGracefulTerminalUsed()
	}

	exec.llmOpts = map[string]any{
		"max_tokens":       ts.agent.MaxTokens,
		"temperature":      ts.agent.Temperature,
		"prompt_cache_key": ts.agent.ID,
	}
	if exec.useNativeSearch {
		exec.llmOpts["native_search"] = true
	}
	// Determine effective thinking level: channel override > agent global
	effectiveThinkingLevel := ts.agent.ThinkingLevel
	if behavior := al.getChannelBehavior(ts.channel); behavior != nil {
		if override := behavior.EffectiveThinkingOverride(); override != "" {
			effectiveThinkingLevel = parseThinkingLevel(override)
		}
	}
	if effectiveThinkingLevel != ThinkingOff {
		if tc, ok := ts.agent.Provider.(providers.ThinkingCapable); ok && tc.SupportsThinking() {
			exec.llmOpts["thinking_level"] = string(effectiveThinkingLevel)
		} else {
			logger.WarnCF("agent", "thinking_level is set but current provider does not support it, ignoring",
				map[string]any{"agent_id": ts.agent.ID, "thinking_level": string(effectiveThinkingLevel)})
		}
	}

	exec.llmModel = exec.activeModel

	// BeforeLLM hook
	if p.Hooks != nil {
		llmReq, decision := p.Hooks.BeforeLLM(turnCtx, &LLMHookRequest{
			Meta:             ts.eventMeta("runTurn", "turn.llm.request"),
			Context:          cloneTurnContext(ts.turnCtx),
			Model:            exec.llmModel,
			Messages:         exec.callMessages,
			Tools:            exec.providerToolDefs,
			Options:          exec.llmOpts,
			GracefulTerminal: exec.gracefulTerminal,
		})
		switch decision.normalizedAction() {
		case HookActionContinue, HookActionModify:
			if llmReq != nil {
				exec.llmModel = llmReq.Model
				exec.callMessages = llmReq.Messages
				exec.providerToolDefs = llmReq.Tools
				exec.llmOpts = llmReq.Options
			}
		case HookActionAbortTurn:
			exec.abortedByHook = true
			return ControlBreak, nil
		case HookActionHardAbort:
			_ = ts.requestHardAbort()
			exec.abortedByHardAbort = true
			return ControlBreak, nil
		}
	}

	al.emitEvent(
		EventKindLLMRequest,
		ts.eventMeta("runTurn", "turn.llm.request"),
		LLMRequestPayload{
			Model:         exec.llmModel,
			MessagesCount: len(exec.callMessages),
			ToolsCount:    len(exec.providerToolDefs),
			MaxTokens:     ts.agent.MaxTokens,
			Temperature:   ts.agent.Temperature,
		},
	)

	logger.DebugCF("agent", "LLM request",
		map[string]any{
			"agent_id":          ts.agent.ID,
			"iteration":         iteration,
			"model":             exec.llmModel,
			"messages_count":    len(exec.callMessages),
			"tools_count":       len(exec.providerToolDefs),
			"max_tokens":        ts.agent.MaxTokens,
			"temperature":       ts.agent.Temperature,
			"system_prompt_len": len(exec.callMessages[0].Content),
		})
	logger.DebugCF("agent", "Full LLM request",
		map[string]any{
			"iteration":     iteration,
			"messages_json": formatMessagesForLog(exec.callMessages),
			"tools_json":    formatToolsForLog(exec.providerToolDefs),
		})

	// LLM call closure with fallback support
	callLLM := func(messagesForCall []providers.Message, toolDefsForCall []providers.ToolDefinition) (*providers.LLMResponse, error) {
		providerCtx, providerCancel := context.WithCancel(turnCtx)
		ts.setProviderCancel(providerCancel)
		defer func() {
			providerCancel()
			ts.clearProviderCancel(providerCancel)
		}()

		al.activeRequests.Add(1)
		defer al.activeRequests.Done()

		if len(exec.activeCandidates) > 1 && p.Fallback != nil {
			fbResult, fbErr := p.Fallback.Execute(
				providerCtx,
				exec.activeCandidates,
				func(ctx context.Context, provider, model string) (*providers.LLMResponse, error) {
					candidateProvider := exec.activeProvider
					if cp, ok := ts.agent.CandidateProviders[providers.ModelKey(provider, model)]; ok {
						candidateProvider = cp
					}
					return candidateProvider.Chat(ctx, messagesForCall, toolDefsForCall, model, exec.llmOpts)
				},
			)
			if fbErr != nil {
				return nil, fbErr
			}
			if fbResult.Provider != "" && len(fbResult.Attempts) > 0 {
				logger.InfoCF(
					"agent",
					fmt.Sprintf("Fallback: succeeded with %s/%s after %d attempts",
						fbResult.Provider, fbResult.Model, len(fbResult.Attempts)+1),
					map[string]any{"agent_id": ts.agent.ID, "iteration": iteration},
				)
			}
			return fbResult.Response, nil
		}
		return exec.activeProvider.Chat(providerCtx, messagesForCall, toolDefsForCall, exec.llmModel, exec.llmOpts)
	}

	// Determine if real-time token flow should be shown
	var tracker *tokenFlowTracker
	var tokenFlowTrackerStarted bool
	if !constants.IsInternalChannel(ts.channel) {
		showTokenFlow := func() bool {
			behavior := al.getChannelBehavior(ts.channel)
			globalDefault := true
			if al.cfg != nil && al.cfg.Agents.Defaults.BehaviorDefaults != nil && al.cfg.Agents.Defaults.BehaviorDefaults.ShowTokenFlow != nil {
				globalDefault = *al.cfg.Agents.Defaults.BehaviorDefaults.ShowTokenFlow
			}
			return behavior.EffectiveShowTokenFlow(globalDefault)
		}()
		if showTokenFlow {
			if _, ok := exec.activeProvider.(providers.StreamingProvider); ok {
				intervalSec := 3
				if behavior := al.getChannelBehavior(ts.channel); behavior != nil && behavior.TokenFlowIntervalSec != nil {
					intervalSec = *behavior.TokenFlowIntervalSec
				} else if al.cfg != nil && al.cfg.Agents.Defaults.BehaviorDefaults != nil && al.cfg.Agents.Defaults.BehaviorDefaults.TokenFlowIntervalSec != nil {
					intervalSec = *al.cfg.Agents.Defaults.BehaviorDefaults.TokenFlowIntervalSec
				}
				tracker = newTokenFlowTracker(al, ts.channel, ts.chatID, intervalSec)
				if tracker.start(turnCtx) {
					tokenFlowTrackerStarted = true
				}
			}
		}
	}

	// Wrap callLLM to inject streaming: on retries the tracker is reused,
	// accumulating chars across attempts for the live display.
	if tokenFlowTrackerStarted {
		originalCallLLM := callLLM
		callLLM = func(messagesForCall []providers.Message, toolDefsForCall []providers.ToolDefinition) (*providers.LLMResponse, error) {
			streamProvider, ok := exec.activeProvider.(providers.StreamingProvider)
			if !ok || len(exec.activeCandidates) > 1 {
				// Fallback path or non-streaming provider: fall through to original.
				return originalCallLLM(messagesForCall, toolDefsForCall)
			}
			providerCtx, providerCancel := context.WithCancel(turnCtx)
			ts.setProviderCancel(providerCancel)
			defer func() {
				providerCancel()
				ts.clearProviderCancel(providerCancel)
			}()

			al.activeRequests.Add(1)
			defer al.activeRequests.Done()

			exec.usedStreaming = true

			return streamProvider.ChatStream(
				providerCtx,
				messagesForCall,
				toolDefsForCall,
				exec.llmModel,
				exec.llmOpts,
				func(accumulated string) {
					tracker.onChunk(accumulated)
				},
			)
		}
	}

	// Retry loop
	var err error
	maxRetries := 2
	for retry := 0; retry <= maxRetries; retry++ {
		exec.response, err = callLLM(exec.callMessages, exec.providerToolDefs)
		if err == nil {
			break
		}
		if ts.hardAbortRequested() && errors.Is(err, context.Canceled) {
			_ = ts.requestHardAbort()
			exec.abortedByHardAbort = true
			return ControlBreak, nil
		}

		// Retry without media if vision is unsupported
		if hasMediaRefs(exec.callMessages) && isVisionUnsupportedError(err) && retry < maxRetries {
			al.emitEvent(
				EventKindLLMRetry,
				ts.eventMeta("runTurn", "turn.llm.retry"),
				LLMRetryPayload{
					Attempt:    retry + 1,
					MaxRetries: maxRetries,
					Reason:     "vision_unsupported",
					Error:      err.Error(),
					Backoff:    0,
				},
			)
			logger.WarnCF("agent", "Vision unsupported, retrying without media", map[string]any{
				"error": err.Error(),
				"retry": retry,
			})
			exec.callMessages = stripMessageMedia(exec.callMessages)
			if !ts.opts.NoHistory {
				exec.history = stripMessageMedia(exec.history)
				ts.agent.Sessions.SetHistory(ts.sessionKey, exec.history)
				for i := range ts.persistedMessages {
					ts.persistedMessages[i].Media = nil
				}
				ts.refreshRestorePointFromSession(ts.agent)
			}
			continue
		}

		errMsg := strings.ToLower(err.Error())
		isTimeoutError := errors.Is(err, context.DeadlineExceeded) ||
			strings.Contains(errMsg, "deadline exceeded") ||
			strings.Contains(errMsg, "client.timeout") ||
			strings.Contains(errMsg, "timed out") ||
			strings.Contains(errMsg, "timeout exceeded")

		isContextError := !isTimeoutError && (strings.Contains(errMsg, "context_length_exceeded") ||
			strings.Contains(errMsg, "context window") ||
			strings.Contains(errMsg, "context_window") ||
			strings.Contains(errMsg, "maximum context length") ||
			strings.Contains(errMsg, "token limit") ||
			strings.Contains(errMsg, "too many tokens") ||
			strings.Contains(errMsg, "max_tokens") ||
			strings.Contains(errMsg, "invalidparameter") ||
			strings.Contains(errMsg, "prompt is too long") ||
			strings.Contains(errMsg, "request too large"))

		if isTimeoutError && retry < maxRetries {
			backoff := time.Duration(retry+1) * 5 * time.Second
			al.emitEvent(
				EventKindLLMRetry,
				ts.eventMeta("runTurn", "turn.llm.retry"),
				LLMRetryPayload{
					Attempt:    retry + 1,
					MaxRetries: maxRetries,
					Reason:     "timeout",
					Error:      err.Error(),
					Backoff:    backoff,
				},
			)
			logger.WarnCF("agent", "Timeout error, retrying after backoff", map[string]any{
				"error":   err.Error(),
				"retry":   retry,
				"backoff": backoff.String(),
			})
			if sleepErr := sleepWithContext(turnCtx, backoff); sleepErr != nil {
				if ts.hardAbortRequested() {
					_ = ts.requestHardAbort()
					return ControlBreak, nil
				}
				err = sleepErr
				break
			}
			continue
		}

		if isContextError && retry < maxRetries && !ts.opts.NoHistory {
			al.emitEvent(
				EventKindLLMRetry,
				ts.eventMeta("runTurn", "turn.llm.retry"),
				LLMRetryPayload{
					Attempt:    retry + 1,
					MaxRetries: maxRetries,
					Reason:     "context_limit",
					Error:      err.Error(),
				},
			)
			logger.WarnCF(
				"agent",
				"Context window error detected, attempting compression",
				map[string]any{
					"error": err.Error(),
					"retry": retry,
				},
			)

			if retry == 0 && !constants.IsInternalChannel(ts.channel) {
				al.bus.PublishOutbound(ctx, outboundMessageForTurn(
					ts,
					"Context window exceeded. Compressing history and retrying...",
				))
			}

			if compactErr := p.ContextManager.Compact(ctx, &CompactRequest{
				SessionKey: ts.sessionKey,
				Reason:     ContextCompressReasonRetry,
				Budget:     ts.agent.ContextWindow,
			}); compactErr != nil {
				logger.WarnCF("agent", "Context overflow compact failed", map[string]any{
					"session_key": ts.sessionKey,
					"error":       compactErr.Error(),
				})
			}
			ts.refreshRestorePointFromSession(ts.agent)
			if asmResp, asmErr := p.ContextManager.Assemble(ctx, &AssembleRequest{
				SessionKey: ts.sessionKey,
				Budget:     ts.agent.ContextWindow,
				MaxTokens:  ts.agent.MaxTokens,
			}); asmErr == nil && asmResp != nil {
				exec.history = asmResp.History
				exec.summary = asmResp.Summary
			}
			exec.messages = ts.agent.ContextBuilder.BuildMessagesFromPrompt(
				promptBuildRequestForTurn(ts, exec.history, exec.summary, "", nil),
			)
			exec.callMessages = exec.messages
			if exec.gracefulTerminal {
				msgs := append([]providers.Message(nil), exec.messages...)
				exec.callMessages = append(msgs, ts.interruptHintMessage())
			}
			continue
		}
		break
	}

	// Finalize token flow tracker: stop the background edit loop and
	// replace the streaming message with final content + token statistics.
	if tokenFlowTrackerStarted && tracker != nil && exec.response != nil {
		content := exec.response.Content
		if content == "" && exec.response.ReasoningContent != "" && ts.channel != "pico" {
			content = exec.response.ReasoningContent
		}
		tracker.done(turnCtx, content, exec.response.Usage)
	}

	if err != nil {
		al.emitEvent(
			EventKindError,
			ts.eventMeta("runTurn", "turn.error"),
			ErrorPayload{
				Stage:   "llm",
				Message: err.Error(),
			},
		)
		logger.ErrorCF("agent", "LLM call failed",
			map[string]any{
				"agent_id":  ts.agent.ID,
				"iteration": iteration,
				"model":     exec.llmModel,
				"error":     err.Error(),
			})
		return ControlBreak, fmt.Errorf("LLM call failed after retries: %w", err)
	}

	// AfterLLM hook
	if p.Hooks != nil {
		llmResp, decision := p.Hooks.AfterLLM(turnCtx, &LLMHookResponse{
			Meta:     ts.eventMeta("runTurn", "turn.llm.response"),
			Context:  cloneTurnContext(ts.turnCtx),
			Model:    exec.llmModel,
			Response: exec.response,
		})
		switch decision.normalizedAction() {
		case HookActionContinue, HookActionModify:
			if llmResp != nil && llmResp.Response != nil {
				exec.response = llmResp.Response
			}
		case HookActionAbortTurn:
			exec.abortedByHook = true
			return ControlBreak, nil
		case HookActionHardAbort:
			_ = ts.requestHardAbort()
			exec.abortedByHardAbort = true
			return ControlBreak, nil
		}
	}

	// Save finishReason to turnState for SubTurn truncation detection
	if innerTS := turnStateFromContext(ctx); innerTS != nil {
		innerTS.SetLastFinishReason(exec.response.FinishReason)
		if exec.response.Usage != nil {
			innerTS.SetLastUsage(exec.response.Usage)
		}
	}

	reasoningContent := responseReasoningContent(exec.response)
	hasReasoning := reasoningContent != ""
	shouldPublishInterimToolCalls := !constants.IsInternalChannel(ts.channel) && len(exec.response.ToolCalls) > 0
	shouldPublishPicoToolCallInterim := shouldPublishInterimToolCalls && ts.channel == "pico"

	// Check channel behavior for ShowReasoning.
	// Default: true (backward compatible — reasoning is shown unless explicitly disabled).
	behavior := al.getChannelBehavior(ts.channel)
	showReasoningDefault := true
	if behavior != nil && behavior.ShowReasoning != nil {
		showReasoningDefault = *behavior.ShowReasoning
	} else if al.cfg.Agents.Defaults.BehaviorDefaults != nil && al.cfg.Agents.Defaults.BehaviorDefaults.ShowReasoning != nil {
		showReasoningDefault = *al.cfg.Agents.Defaults.BehaviorDefaults.ShowReasoning
	}
	showReasoning := hasReasoning && showReasoningDefault

	if shouldPublishPicoToolCallInterim {
		// Pico tool-call turns publish their reasoning/content/tool summary as a
		// structured sequence after the tool-call payload is normalized below.
	} else if shouldPublishInterimToolCalls {
		// Non-pico channels: still publish, go via handleReasoning for the interim
		// content, and structured tool-call details will be published after normalization.
		if showReasoning {
			go al.handleReasoning(turnCtx, reasoningContent, ts.channel, al.targetReasoningChannelID(ts.channel))
		}
	} else if ts.channel == "pico" {
		if showReasoning {
			go al.publishPicoReasoning(turnCtx, reasoningContent, ts.chatID)
		}
	} else {
		if showReasoning {
			go al.handleReasoning(
				turnCtx,
				reasoningContent,
				ts.channel,
				al.targetReasoningChannelID(ts.channel),
			)
		}
	}
	al.emitEvent(
		EventKindLLMResponse,
		ts.eventMeta("runTurn", "turn.llm.response"),
		LLMResponsePayload{
			ContentLen:   len(exec.response.Content),
			ToolCalls:    len(exec.response.ToolCalls),
			HasReasoning: exec.response.Reasoning != "" || exec.response.ReasoningContent != "",
		},
	)

	llmResponseFields := map[string]any{
		"agent_id":       ts.agent.ID,
		"iteration":      iteration,
		"content_chars":  len(exec.response.Content),
		"tool_calls":     len(exec.response.ToolCalls),
		"reasoning":      exec.response.Reasoning,
		"target_channel": al.targetReasoningChannelID(ts.channel),
		"channel":        ts.channel,
	}
	if exec.response.Usage != nil {
		llmResponseFields["prompt_tokens"] = exec.response.Usage.PromptTokens
		llmResponseFields["completion_tokens"] = exec.response.Usage.CompletionTokens
		llmResponseFields["total_tokens"] = exec.response.Usage.TotalTokens
	}
	logger.DebugCF("agent", "LLM response", llmResponseFields)

	// No-tool-call path: steering check and direct response
	if len(exec.response.ToolCalls) == 0 || exec.gracefulTerminal {
		responseContent := exec.response.Content
		if responseContent == "" && exec.response.ReasoningContent != "" && ts.channel != "pico" {
			responseContent = exec.response.ReasoningContent
		}
		if steerMsgs := al.dequeueSteeringMessagesForScope(ts.sessionKey); len(steerMsgs) > 0 {
			logger.InfoCF("agent", "Steering arrived after direct LLM response; continuing turn",
				map[string]any{
					"agent_id":       ts.agent.ID,
					"iteration":      iteration,
					"steering_count": len(steerMsgs),
				})
			exec.pendingMessages = append(exec.pendingMessages, steerMsgs...)
			return ControlContinue, nil
		}
		exec.finalContent = responseContent
		logger.InfoCF("agent", "LLM response without tool calls (direct answer)",
			map[string]any{
				"agent_id":      ts.agent.ID,
				"iteration":     iteration,
				"content_chars": len(exec.finalContent),
			})
		return ControlBreak, nil
	}

	// Tool-call path: normalize and prepare for tool execution
	exec.normalizedToolCalls = make([]providers.ToolCall, 0, len(exec.response.ToolCalls))
	for _, tc := range exec.response.ToolCalls {
		exec.normalizedToolCalls = append(exec.normalizedToolCalls, providers.NormalizeToolCall(tc))
	}

	toolNames := make([]string, 0, len(exec.normalizedToolCalls))
	for _, tc := range exec.normalizedToolCalls {
		toolNames = append(toolNames, tc.Name)
	}
	logger.InfoCF("agent", "LLM requested tool calls",
		map[string]any{
			"agent_id":  ts.agent.ID,
			"tools":     toolNames,
			"count":     len(exec.normalizedToolCalls),
			"iteration": iteration,
		})

	exec.allResponsesHandled = len(exec.normalizedToolCalls) > 0
	assistantMsg := providers.Message{
		Role:             "assistant",
		Content:          exec.response.Content,
		ReasoningContent: reasoningContent,
	}
	for _, tc := range exec.normalizedToolCalls {
		argumentsJSON, _ := json.Marshal(tc.Arguments)
		toolFeedbackExplanation := toolFeedbackExplanationForToolCall(
			exec.response,
			tc,
			exec.messages,
		)
		extraContent := tc.ExtraContent
		if strings.TrimSpace(toolFeedbackExplanation) != "" {
			if extraContent == nil {
				extraContent = &providers.ExtraContent{}
			}
			extraContent.ToolFeedbackExplanation = toolFeedbackExplanation
		}
		thoughtSignature := ""
		if tc.Function != nil {
			thoughtSignature = tc.Function.ThoughtSignature
		}
		assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, providers.ToolCall{
			ID:   tc.ID,
			Type: "function",
			Name: tc.Name,
			Function: &providers.FunctionCall{
				Name:             tc.Name,
				Arguments:        string(argumentsJSON),
				ThoughtSignature: thoughtSignature,
			},
			ExtraContent:     extraContent,
			ThoughtSignature: thoughtSignature,
		})
	}
	exec.messages = append(exec.messages, assistantMsg)
	if !ts.opts.NoHistory {
		ts.agent.Sessions.AddFullMessage(ts.sessionKey, assistantMsg)
		ts.recordPersistedMessage(assistantMsg)
		ts.ingestMessage(turnCtx, al, assistantMsg)
	}
	if shouldPublishPicoToolCallInterim {
		al.publishPicoToolCallInterim(
			turnCtx,
			ts,
			reasoningContent,
			exec.response.Content,
			assistantMsg.ToolCalls,
		)
	} else if shouldPublishInterimToolCalls {
		// Non-pico channels: publish reasoning and tool-call summary for visibility
		if showReasoning {
			go al.handleReasoning(turnCtx, reasoningContent, ts.channel, al.targetReasoningChannelID(ts.channel))
		}
		if strings.TrimSpace(exec.response.Content) != "" {
			go func() {
				pubCtx, pubCancel := context.WithTimeout(turnCtx, 3*time.Second)
				defer pubCancel()
				_ = al.bus.PublishOutbound(pubCtx, outboundMessageForTurn(ts, exec.response.Content))
			}()
		}
	}

	return ControlToolLoop, nil
}

// === ThinkingLevel types and parsing (from thinking.go) ===

// ThinkingLevel controls how the provider sends thinking parameters.
//
//   - "adaptive": sends {thinking: {type: "adaptive"}} + output_config.effort (Claude 4.6+)
//   - "low"/"medium"/"high"/"xhigh": sends {thinking: {type: "enabled", budget_tokens: N}} (all models)
//   - "off": disables thinking
type ThinkingLevel string

const (
	ThinkingOff      ThinkingLevel = "off"
	ThinkingLow      ThinkingLevel = "low"
	ThinkingMedium   ThinkingLevel = "medium"
	ThinkingHigh     ThinkingLevel = "high"
	ThinkingXHigh    ThinkingLevel = "xhigh"
	ThinkingAdaptive ThinkingLevel = "adaptive"
)

// parseThinkingLevel normalizes a config string to a ThinkingLevel.
// Case-insensitive and whitespace-tolerant for user-facing config values.
// Returns ThinkingOff for unknown or empty values.
func parseThinkingLevel(level string) ThinkingLevel {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "adaptive":
		return ThinkingAdaptive
	case "low":
		return ThinkingLow
	case "medium":
		return ThinkingMedium
	case "high":
		return ThinkingHigh
	case "xhigh":
		return ThinkingXHigh
	default:
		return ThinkingOff
	}
}

// === Media helpers for LLM fallback (from llm_media.go) ===

func messagesContainMedia(messages []providers.Message) bool {
	for _, msg := range messages {
		for _, ref := range msg.Media {
			if strings.TrimSpace(ref) != "" {
				return true
			}
		}
	}
	return false
}

func stripMessageMedia(messages []providers.Message) []providers.Message {
	if !messagesContainMedia(messages) {
		return messages
	}
	stripped := make([]providers.Message, len(messages))
	for i, msg := range messages {
		stripped[i] = msg
		stripped[i].Media = nil
	}
	return stripped
}

func isVisionUnsupportedError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "no endpoints found that support image input") {
		return true
	}
	if strings.Contains(msg, "does not support image input") ||
		strings.Contains(msg, "does not support image inputs") ||
		strings.Contains(msg, "does not support images") ||
		strings.Contains(msg, "image input is not supported") ||
		strings.Contains(msg, "images are not supported") ||
		strings.Contains(msg, "does not support vision") ||
		strings.Contains(msg, "unsupported content type: image_url") {
		return true
	}
	if strings.Contains(msg, "image_url") && strings.Contains(msg, "invalid") {
		return true
	}
	return false
}
