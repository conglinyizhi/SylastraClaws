// SylastraClaws - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 SylastraClaws contributors

package channels

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/conglinyizhi/SylastraClaws/pkg/bus"
	"github.com/conglinyizhi/SylastraClaws/pkg/config"
	"github.com/conglinyizhi/SylastraClaws/pkg/media"
)

const (
	defaultChannelQueueSize = 16
	defaultRateLimit        = 10 // default 10 msg/s
	maxRetries              = 3
	rateLimitDelay          = 1 * time.Second
	baseBackoff             = 500 * time.Millisecond
	maxBackoff              = 8 * time.Second

	janitorInterval = 10 * time.Second
	typingStopTTL   = 5 * time.Minute
	placeholderTTL  = 10 * time.Minute
)

type typingEntry struct {
	stop      func()
	createdAt time.Time
}

type reactionEntry struct {
	undo      func()
	createdAt time.Time
}

type placeholderEntry struct {
	id        string
	createdAt time.Time
}

var channelRateConfig = map[string]float64{
	"telegram": 20,
	"discord":  1,
	"slack":    1,
	"matrix":   2,
	"line":     10,
	"qq":       5,
	"irc":      2,
}

type channelWorker struct {
	ch         Channel
	queue      chan bus.OutboundMessage
	mediaQueue chan bus.OutboundMediaMessage
	done       chan struct{}
	mediaDone  chan struct{}
	limiter    *rate.Limiter
}

type Manager struct {
	channels      map[string]Channel
	workers       map[string]*channelWorker
	bus           *bus.MessageBus
	config        *config.Config
	mediaStore    media.MediaStore
	dispatchTask  *asyncTask
	mux           *dynamicServeMux
	httpServer    *http.Server
	httpListeners []net.Listener
	mu            sync.RWMutex
	placeholders  sync.Map
	typingStops   sync.Map
	reactionUndos sync.Map
	streamActive  sync.Map
	channelHashes map[string]string
}

type toolFeedbackMessageTracker interface {
	RecordToolFeedbackMessage(chatID, messageID, content string)
	ClearToolFeedbackMessage(chatID string)
}

type toolFeedbackMessageCleaner interface {
	DismissToolFeedbackMessage(ctx context.Context, chatID string)
}

type toolFeedbackMessageTargetResolver interface {
	ToolFeedbackMessageChatID(chatID string, outboundCtx *bus.InboundContext) string
}

type toolFeedbackMessageContentPreparer interface {
	PrepareToolFeedbackMessageContent(content string) string
}

type asyncTask struct {
	cancel context.CancelFunc
}
