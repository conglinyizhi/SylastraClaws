# Test File Analysis: Splitting Feasibility

## Files Analyzed
- `agent_test.go` (5583 lines)
- `steering_test.go` (1555 lines)

## Test Counts
- **agent_test.go**: 50 top-level `func Test` + 27 subtests = ~77 test cases
- **steering_test.go**: 21 top-level `func Test` + 4 subtests = ~25 test cases
- **Total**: ~102 test cases (task states 121 — some subtests may be counted differently)

---

## Category Classification

### Category 1: TRULY ISOLATED — Pure functions, no global state, no NewAgentLoop, no disk I/O

These tests can be cleanly split into a `pkg/agent/agenttest/` subpackage.

#### steering_test.go (7 tests)
| Test | Reason |
|------|--------|
| `TestSteeringQueue_PushDequeue_OneAtATime` | Pure queue struct test |
| `TestSteeringQueue_PushDequeue_All` | Pure queue struct test |
| `TestSteeringQueue_EmptyDequeue` | Pure queue struct test |
| `TestSteeringQueue_SetMode` | Pure queue struct test |
| `TestSteeringQueue_ConcurrentAccess` | Pure queue struct test (sync only) |
| `TestSteeringQueue_Overflow` | Pure queue struct test |
| `TestParseSteeringMode` + 4 subtests | Pure function test |

#### agent_test.go (23 tests)
| Test | Reason |
|------|--------|
| `TestToolContext_Updates` | Pure context helpers, no AgentLoop |
| `TestToolFeedbackExplanationFromResponse_UsesCurrentContentFirst` | Pure function |
| `TestSideQuestionResponseContent_FallsBackWhenContentIsWhitespace` | Pure function |
| `TestResponseReasoningContent_FallsBackWhenReasoningIsWhitespace` | Pure function |
| `TestToolFeedbackExplanationFromResponse_UsesExplicitToolCallExtraContent` | Pure function |
| `TestToolFeedbackExplanationForToolCall_PrefersToolSpecificExtraContent` | Pure function |
| `TestToolFeedbackExplanationForToolCall_DoesNotReuseAnotherToolCallExplanation` | Pure function |
| `TestToolFeedbackExplanationFromResponse_DoesNotUseReasoningContent` | Pure function |
| `TestToolFeedbackExplanationForToolCall_DoesNotTruncateLongExplanation` | Pure function |
| `TestToolFeedbackArgsPreview_UsesJSONAndTruncates` | Pure function |
| `TestIsNativeSearchProvider_Supported` | Pure interface check |
| `TestIsNativeSearchProvider_NotSupported` | Pure interface check |
| `TestIsNativeSearchProvider_NoInterface` | Pure interface check |
| `TestFilterClientWebSearch_RemovesWebSearch` | Pure function |
| `TestFilterClientWebSearch_NoWebSearch` | Pure function |
| `TestFilterClientWebSearch_EmptyInput` | Pure function |
| `TestResolveMediaRefs_PassesThroughNonMediaRefs` | Pure function (nil store) |
| `TestInjectPathTags_HandlesVariousChannelPlaceholders` + 9 subtests | Pure function |
| `TestInjectPathTags_DoesNotReplacePathTag` | Pure function |
| `TestInjectPathTags_PrependsForJSONContent` | Pure function |
| `TestInjectPathTags_BracketTextNotTreatedAsJSON` | Pure function |

**Subtotal: 30 tests** (7 + 23) — approximately **25% of all tests**

---

### Category 2: HAS DISK I/O but NO AgentLoop — Could be split with store mock

These use `t.TempDir()` + `media.NewFileMediaStore()` + `resolveMediaRefs()`, but DON'T use `NewAgentLoop`.

#### agent_test.go (11 tests)
| Test | Dependencies |
|------|-------------|
| `TestResolveMediaRefs_ImageInjectsPathTag` | FileMediaStore, disk I/O |
| `TestResolveMediaRefs_ToolRoleImageAppendedAsUserMessage` | FileMediaStore, disk I/O |
| `TestResolveMediaRefs_MultiToolCallPreservesOrdering` | FileMediaStore, disk I/O |
| `TestResolveMediaRefs_OversizedImageSkipsBase64KeepsPathTag` | FileMediaStore, disk I/O |
| `TestResolveMediaRefs_UnknownTypeInjectsPath` | FileMediaStore, disk I/O |
| `TestResolveMediaRefs_DoesNotMutateOriginal` | FileMediaStore, disk I/O |
| `TestResolveMediaRefs_UsesMetaContentType` | FileMediaStore, disk I/O |
| `TestResolveMediaRefs_PDFInjectsFilePath` | FileMediaStore, disk I/O |
| `TestResolveMediaRefs_AudioInjectsAudioPath` | FileMediaStore, disk I/O |
| `TestResolveMediaRefs_VideoInjectsVideoPath` | FileMediaStore, disk I/O |
| `TestResolveMediaRefs_NoGenericTagAppendsPath` | FileMediaStore, disk I/O |
| `TestResolveMediaRefs_JSONContentPrependsPathTag` | FileMediaStore, disk I/O |
| `TestResolveMediaRefs_EmptyContentGetsPathTag` | FileMediaStore, disk I/O |
| `TestResolveMediaRefs_MixedImageAndFile` | FileMediaStore, disk I/O |

**Subtotal: 14 tests** — These test `resolveMediaRefs()` which is an exported? function. Could be split with an interface-based store mock instead of FileMediaStore, but requires refactoring.

---

### Category 3: HEAVY INTEGRATION — Uses NewAgentLoop (requires registry + provider + bus)

These tests constitute the majority and must stay in the main package.

#### agent_test.go (~36 tests)
`TestNewAgentLoop_RegistersWebSearchTool`, `TestNewAgentLoop_RegistersWebSearchTool_WhenExplicitProviderUnavailable`, `TestNewAgentLoop_DoesNotRegisterWebSearchTool_WhenNoReadyProviders`, `TestProcessMessage_IncludesCurrentSenderInDynamicContext`, `TestProcessMessage_UseCommandLoadsRequestedSkill`, `TestProcessMessage_BtwCommandRunsWithoutPersistingHistory`, `TestProcessMessage_BtwCommandIncludesRequestContextAndMedia`, `TestProcessMessage_BtwCommandUsesIsolatedProvider`, `TestProcessMessage_BtwCommandRetriesWithoutMediaOnVisionUnsupported`, `TestProcessMessage_BtwCommandUsesProviderFactoryModel`, `TestProcessMessage_BtwCommandHookModelBypassesFallbackCandidates`, `TestHandleCommand_UseCommandRejectsUnknownSkill`, `TestProcessMessage_UseCommandArmsSkillForNextMessage`, `TestApplyExplicitSkillCommand_ArmsSkillForNextMessage`, `TestApplyExplicitSkillCommand_InlineMessageMutatesOptions`, `TestRecordLastChannel`, `TestRecordLastChatID`, `TestNewAgentLoop_StateInitialized`, `TestToolRegistry_ToolRegistration`, `TestToolRegistry_GetDefinitions`, `TestProcessMessage_MediaToolHandledSkipsFollowUpLLMAndFinalText`, `TestProcessMessage_HandledToolProcessesQueuedSteeringBeforeReturning`, `TestRunAgentLoop_ResponseHandledToolPublishesForUserWhenSendResponseDisabled`, `TestAppendEventContextFields_IncludesInboundRouteAndScope`, `TestResolveMessageRoute_UsesInboundContextAccount`, `TestResolveMessageRoute_UsesDispatchRulesInOrder`, `TestProcessMessage_MediaArtifactCanBeForwardedBySendFile`, `TestAgentLoop_GetStartupInfo`, `TestAgentLoop_Stop`, `TestProcessMessage_UsesRouteSessionKey`, `TestProcessMessage_CommandOutcomes`, `TestProcessMessage_MCPCommandsHandledWithoutLLMCall`, `TestProcessMessage_SwitchModelShowModelConsistency`, `TestProcessMessage_SwitchModelRejectsUnknownAlias`, `TestProcessMessage_SwitchModelRoutesSubsequentRequestsToSelectedProvider`, `TestProcessMessage_ModelRoutingUsesLightProvider`, `TestProcessMessage_FallbackUsesPerCandidateProvider`, `TestProcessMessage_FallbackUsesActiveProviderWhenCandidateNotRegistered`, `TestToolResult_SilentToolDoesNotSendUserMessage`, `TestToolResult_UserFacingToolDoesSendMessage`, `TestAgentLoop_ContextExhaustionRetry`, `TestAgentLoop_VisionUnsupportedErrorStripsSessionMedia`, `TestAgentLoop_EmptyModelResponseUsesAccurateFallback`, `TestAgentLoop_ToolLimitUsesDedicatedFallback`, `TestProcessDirectWithChannel_TriggersMCPInitialization`, `TestTargetReasoningChannelID_AllChannels` + 13 subtests, `TestHandleReasoning` + 5 subtests, `TestProcessMessage_PublishesReasoningContentToReasoningChannel`, `TestProcessMessage_PicoPublishesReasoningAsThoughtMessage`, `TestProcessHeartbeat_DoesNotPublishToolFeedback`, `TestProcessMessage_PublishesToolFeedbackWhenEnabled`, `TestProcessMessage_PersistsReasoningContentInSessionHistory`, `TestProcessMessage_PersistsReasoningToolResponseAsSingleAssistantRecord`, `TestProcessMessage_DoesNotLeakReasoningContentInToolFeedback`, `TestProcessMessage_DoesNotPublishToolFeedbackForDiscordWhenDisabled`, `TestProcessMessage_DoesNotPublishToolFeedbackForTelegramWhenDisabled`, `TestProcessMessage_DoesNotPublishToolFeedbackForFeishuWhenDisabled`, `TestProcessMessage_MessageToolPublishesOutboundWithTurnMetadata`, `TestRun_PicoPublishesAssistantContentDuringToolCallsWithoutFinalDuplicate`, `TestRunAgentLoop_PicoSkipsInterimPublishWhenNotAllowed`, `TestRun_PicoToolFeedbackSuppressesDuplicateInterimAssistantContent`, `TestProcessMessage_ContextOverflowRecovery`, `TestProcessMessage_ContextOverflow_AnthropicStyle`, `TestParallelMessageProcessing_DifferentSessionsProcessedConcurrently`, `TestParallelMessageProcessing_SameSessionProcessedSequentially`

**Subtotal: ~58 tests** (including subtests)

#### steering_test.go (14 tests)
All tests using `newTestAgentLoop` or manual `NewAgentLoop` + `TempDir`:
`TestAgentLoop_Steer_Enqueues`, `TestAgentLoop_SteeringMode_GetSet`, `TestAgentLoop_SteeringMode_ConfiguredFromConfig`, `TestAgentLoop_Continue_NoMessages`, `TestAgentLoop_Continue_WithMessages`, `TestAgentLoop_Steering_SkipsRemainingTools`, `TestAgentLoop_Steering_InitialPoll`, `TestAgentLoop_Run_AutoContinuesLateSteeringMessage`, `TestAgentLoop_Steering_DirectResponseContinuesWithQueuedMessage`, `TestAgentLoop_AgentForSession_UsesStoredScopeMetadata`, `TestAgentLoop_Continue_PreservesSteeringMedia`, `TestAgentLoop_InterruptGraceful_UsesTerminalNoToolCall`, `TestAgentLoop_InterruptHard_RestoresSession`, `TestAgentLoop_Steering_SkippedToolsHaveErrorResults`

**Subtotal: 14 tests**

---

## Summary Counts

| Category | Count | Percentage |
|----------|-------|------------|
| **Truly Isolated** (Category 1) | **30** | ~29% |
| **Disk I/O only** (Category 2) | **14** | ~14% |
| **Heavy Integration** (Category 3) | **~58** | ~57% |

---

## Recommendation

### Option C — Split out a subpackage: ⚠️ MODERATELY WORTHWHILE

**What can be cleanly split:**

The **30 truly isolated tests** (Category 1) can be cleanly moved to `pkg/agent/agenttest/`. These tests:
- Import only `testing`, `providers`, `tools`, `utils` packages
- Test pure functions like `toolFeedbackExplanationFromResponse()`, `sideQuestionResponseContent()`, `responseReasoningContent()`, `isNativeSearchProvider()`, `filterClientWebSearch()`, `injectPathTags()`, `parseSteeringMode()`, and `steeringQueue` methods
- Have zero external dependencies

**What's borderline (Category 2):**

The **14 `resolveMediaRefs` tests** use `FileMediaStore` (disk I/O via `t.TempDir()`). These could be split if `resolveMediaRefs` is refactored to accept a store interface rather than a concrete `FileMediaStore`, or these could be kept alongside the unit tests but require `t.TempDir()`. They're fast (<1ms), but have a disk I/O dependency.

**Verdict:**

**Option A (status quo)** is simpler but misses an opportunity.

**Option C is worth the complexity** specifically because:
1. The steering queue tests are excellent, isolated unit tests that currently live alongside heavy integration tests like `TestParallelMessageProcessing_DifferentSessionsProcessedConcurrently` (which starts a full agent loop with event bus and goroutines)
2. The toolbar feedback explanation tests (~10 tests) are pure functions that test edge cases that are hard to debug within the heavy integration context
3. The `injectPathTags` and `filterClientWebSearch` tests are also pure functions
4. The 30 tests represent ~29% of all tests — a meaningful chunk

**Suggested approach:**
1. Create `pkg/agent/agenttest/` subpackage
2. Move Category 1 (30 tests) there — they need only `providers.Message`, `providers.LLMResponse`, `tools.ToolResult` types which are already exported
3. Optionally move Category 2 (14 tests) if `resolveMediaRefs` is exported or refactored
4. Left Category 3 (~58 tests) stay in main `agent` package
5. All mock types (recordingProvider, simpleMockProvider, etc.) stay in the main package's test files — the unit tests in agenttest/ define their own small mocks if needed

**Risk:** The 30 tests reference internal functions like `toolFeedbackExplanationFromResponse`, `sideQuestionResponseContent`, `responseReasoningContent`, `isNativeSearchProvider`, `filterClientWebSearch`, `injectPathTags`, `newSteeringQueue`, `parseSteeringMode`, etc. These would need to be exported or the tests must use the exported API surface. Currently most are unexported.
