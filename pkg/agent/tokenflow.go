// SylastraClaws - Ultra-lightweight personal AI agent

package agent

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/conglinyizhi/SylastraClaws/pkg/bus"
	"github.com/conglinyizhi/SylastraClaws/pkg/logger"
	"github.com/conglinyizhi/SylastraClaws/pkg/providers"
)

// tokenFlowTracker manages real-time token throughput statistics
// displayed as an editable message during LLM streaming.
//
// Lifecycle:
//  1. New -> startStreamingMessage(ctx) sends a placeholder and records msgID
//  2. OnChunk(accumulated) called on each text delta from the LLM
//  3. Streaming progresses: a background goroutine edits the message at intervals
//  4. Done(ctx, usage):
//     - stops the background goroutine
//     - edits the message one last time with final token statistics
//     - final content is the LLM's text output suffixed with the stats bar
type tokenFlowTracker struct {
	al          *AgentLoop
	channelName string
	chatID      string
	intervalSec int

	mu          sync.Mutex
	streamer    bus.Streamer
	totalChars  int64
	chunkCount  int64
	cancelLoop  context.CancelFunc
	loopStarted atomic.Bool
	started     time.Time
}

func newTokenFlowTracker(al *AgentLoop, channelName, chatID string, intervalSec int) *tokenFlowTracker {
	if intervalSec < 1 {
		intervalSec = 3
	}
	return &tokenFlowTracker{
		al:          al,
		channelName: channelName,
		chatID:      chatID,
		intervalSec: intervalSec,
	}
}

// start begins the token flow display by acquiring a Streamer from the
// channel manager. Returns false if streaming is unavailable (no-op degrade).
func (t *tokenFlowTracker) start(ctx context.Context) bool {
	if t.al.channelManager == nil {
		return false
	}
	streamer, ok := t.al.channelManager.GetStreamer(ctx, t.channelName, t.chatID)
	if !ok {
		return false
	}
	t.mu.Lock()
	t.streamer = streamer
	t.started = time.Now()
	t.mu.Unlock()

	// Start a background goroutine that periodically updates the streaming message.
	editCtx, cancel := context.WithCancel(context.Background())
	t.cancelLoop = cancel
	t.loopStarted.Store(true)
	go t.editLoop(editCtx)
	return true
}

// onChunk is called on every text delta from the LLM.
func (t *tokenFlowTracker) onChunk(accumulated string) {
	t.mu.Lock()
	t.totalChars = int64(len(accumulated))
	t.chunkCount++
	t.mu.Unlock()
}

// done finalizes the token flow message with final token statistics and
// the full assistant content.
func (t *tokenFlowTracker) done(ctx context.Context, content string, usage *providers.UsageInfo) {
	t.cancelLoop()

	t.mu.Lock()
	s := t.streamer
	elapsed := time.Since(t.started)
	chars := t.totalChars
	t.mu.Unlock()

	if s == nil {
		return
	}

	finalContent := content
	stats := buildTokenFlowFinalStats(chars, elapsed, usage)
	if stats != "" {
		finalContent = content + "\n\n" + stats
	}

	if fErr := s.Finalize(ctx, finalContent); fErr != nil {
		logger.WarnCF("agent", "TokenFlow: Finalize failed",
			map[string]any{
				"channel": t.channelName,
				"error":   fErr.Error(),
			})
	}
}

// editLoop runs in the background and updates the streaming message at regular intervals.
func (t *tokenFlowTracker) editLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(t.intervalSec) * time.Second)
	defer ticker.Stop()
	defer t.loopStarted.Store(false)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		t.mu.Lock()
		s := t.streamer
		chars := t.totalChars
		elapsed := time.Since(t.started)
		t.mu.Unlock()

		if s == nil {
			continue
		}

		stats := buildTokenFlowLiveStats(chars, elapsed)
		if stats == "" {
			continue
		}

		if uErr := s.Update(ctx, stats); uErr != nil {
			// One failure: log and stop the loop (next edit would likely fail too).
			logger.DebugCF("agent", "TokenFlow: Update failed, stopping",
				map[string]any{
					"channel": t.channelName,
					"error":   uErr.Error(),
				})
			return
		}
	}
}

// buildTokenFlowLiveStats builds the statistics bar shown while streaming is in progress.
func buildTokenFlowLiveStats(chars int64, elapsed time.Duration) string {
	if chars == 0 {
		return ""
	}
	elapsedSec := elapsed.Seconds()
	rate := float64(chars) / elapsedSec
	return fmt.Sprintf("[📝 %d chars | ⚡ %.0f c/s | ⏱ %.1fs]", chars, rate, elapsedSec)
}

// buildTokenFlowFinalStats builds the final statistics bar shown after streaming completes.
func buildTokenFlowFinalStats(chars int64, elapsed time.Duration, usage *providers.UsageInfo) string {
	elapsedSec := elapsed.Seconds()

	if usage == nil {
		return fmt.Sprintf("[📝 %d chars | ⏱ %.1fs]", chars, elapsedSec)
	}

	input := usage.PromptTokens
	output := usage.CompletionTokens

	if input == 0 && output == 0 {
		return fmt.Sprintf("[📝 %d chars | ⏱ %.1fs]", chars, elapsedSec)
	}

	return fmt.Sprintf("[📥 %d | 📤 %d | ⏱ %.1fs]", input, output, elapsedSec)
}
