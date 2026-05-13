package interfaces

import (
	"context"
	"errors"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
)

// ---- Mock implementations ----

type mockMessageBus struct {
	publishInboundFn      func(ctx context.Context, msg bus.InboundMessage) error
	publishOutboundFn     func(ctx context.Context, msg bus.OutboundMessage) error
	publishOutboundMediaFn func(ctx context.Context, msg bus.OutboundMediaMessage) error
	inboundChanFn         func() <-chan bus.InboundMessage
}

func (m *mockMessageBus) PublishInbound(ctx context.Context, msg bus.InboundMessage) error {
	return m.publishInboundFn(ctx, msg)
}
func (m *mockMessageBus) PublishOutbound(ctx context.Context, msg bus.OutboundMessage) error {
	return m.publishOutboundFn(ctx, msg)
}
func (m *mockMessageBus) PublishOutboundMedia(ctx context.Context, msg bus.OutboundMediaMessage) error {
	return m.publishOutboundMediaFn(ctx, msg)
}
func (m *mockMessageBus) InboundChan() <-chan bus.InboundMessage {
	return m.inboundChanFn()
}

type mockChannelManager struct {
	getChannelFn           func(name string) (channels.Channel, bool)
	getEnabledChannelsFn   func() []string
	invokeTypingStopFn     func(channel, chatID string)
	sendMessageFn          func(ctx context.Context, msg bus.OutboundMessage) error
	sendMediaFn            func(ctx context.Context, msg bus.OutboundMediaMessage) error
	sendPlaceholderFn      func(ctx context.Context, channel, chatID string) bool
	dismissToolFeedbackFn  func(ctx context.Context, channel, chatID string, outboundCtx *bus.InboundContext)
}

func (m *mockChannelManager) GetChannel(name string) (channels.Channel, bool) {
	return m.getChannelFn(name)
}
func (m *mockChannelManager) GetEnabledChannels() []string {
	return m.getEnabledChannelsFn()
}
func (m *mockChannelManager) InvokeTypingStop(channel, chatID string) {
	m.invokeTypingStopFn(channel, chatID)
}
func (m *mockChannelManager) SendMessage(ctx context.Context, msg bus.OutboundMessage) error {
	return m.sendMessageFn(ctx, msg)
}
func (m *mockChannelManager) SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) error {
	return m.sendMediaFn(ctx, msg)
}
func (m *mockChannelManager) SendPlaceholder(ctx context.Context, channel, chatID string) bool {
	return m.sendPlaceholderFn(ctx, channel, chatID)
}
func (m *mockChannelManager) DismissToolFeedback(ctx context.Context, channel, chatID string, outboundCtx *bus.InboundContext) {
	m.dismissToolFeedbackFn(ctx, channel, chatID, outboundCtx)
}

// ---- Tests ----

func TestMessageBusInterface(t *testing.T) {
	t.Run("PublishInbound can be called and returns error", func(t *testing.T) {
		expectedErr := errors.New("test error")
		mb := &mockMessageBus{
			publishInboundFn: func(ctx context.Context, msg bus.InboundMessage) error {
				if msg.Content == "" {
					t.Error("expected non-empty content")
				}
				return expectedErr
			},
		}
		var iface MessageBus = mb // compile-time check
		err := iface.PublishInbound(context.Background(), bus.InboundMessage{Content: "test"})
		if err != expectedErr {
			t.Errorf("got %v, want %v", err, expectedErr)
		}
	})

	t.Run("PublishOutbound can be called with valid message", func(t *testing.T) {
		mb := &mockMessageBus{
			publishOutboundFn: func(ctx context.Context, msg bus.OutboundMessage) error {
				if msg.Channel != "telegram" {
					t.Errorf("channel = %q", msg.Channel)
				}
				return nil
			},
		}
		var iface MessageBus = mb
		err := iface.PublishOutbound(context.Background(), bus.OutboundMessage{Channel: "telegram", Content: "hi"})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("PublishOutboundMedia can be called with parts", func(t *testing.T) {
		mb := &mockMessageBus{
			publishOutboundMediaFn: func(ctx context.Context, msg bus.OutboundMediaMessage) error {
				if len(msg.Parts) == 0 {
					t.Error("expected non-empty parts")
				}
				return nil
			},
		}
		var iface MessageBus = mb
		err := iface.PublishOutboundMedia(context.Background(), bus.OutboundMediaMessage{
			Parts: []bus.MediaPart{{Type: "image"}},
		})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("InboundChan returns a channel", func(t *testing.T) {
		ch := make(chan bus.InboundMessage, 1)
		mb := &mockMessageBus{
			inboundChanFn: func() <-chan bus.InboundMessage {
				return ch
			},
		}
		var iface MessageBus = mb
		got := iface.InboundChan()
		if got == nil {
			t.Fatal("InboundChan returned nil")
		}
		// Verify we can send and receive
		ch <- bus.InboundMessage{Content: "ping"}
		msg := <-got
		if msg.Content != "ping" {
			t.Errorf("got %q, want %q", msg.Content, "ping")
		}
	})
}

func TestChannelManagerInterface(t *testing.T) {
	t.Run("GetChannel returns a channel", func(t *testing.T) {
		cm := &mockChannelManager{
			getChannelFn: func(name string) (channels.Channel, bool) {
				if name == "telegram" {
					return nil, true
				}
				return nil, false
			},
		}
		var iface ChannelManager = cm
		_, ok := iface.GetChannel("telegram")
		if !ok {
			t.Error("GetChannel(telegram) should be found")
		}
		_, ok = iface.GetChannel("nonexistent")
		if ok {
			t.Error("GetChannel(nonexistent) should not be found")
		}
	})

	t.Run("GetEnabledChannels returns list", func(t *testing.T) {
		expected := []string{"telegram", "discord"}
		cm := &mockChannelManager{
			getEnabledChannelsFn: func() []string {
				return expected
			},
		}
		var iface ChannelManager = cm
		got := iface.GetEnabledChannels()
		if len(got) != 2 || got[0] != "telegram" || got[1] != "discord" {
			t.Errorf("got %v, want %v", got, expected)
		}
	})

	t.Run("InvokeTypingStop calls through", func(t *testing.T) {
		called := false
		cm := &mockChannelManager{
			invokeTypingStopFn: func(channel, chatID string) {
				called = true
				if channel != "telegram" || chatID != "123" {
					t.Errorf("InvokeTypingStop(%q, %q)", channel, chatID)
				}
			},
		}
		var iface ChannelManager = cm
		iface.InvokeTypingStop("telegram", "123")
		if !called {
			t.Error("InvokeTypingStop was not called")
		}
	})

	t.Run("SendMessage delegates correctly", func(t *testing.T) {
		cm := &mockChannelManager{
			sendMessageFn: func(ctx context.Context, msg bus.OutboundMessage) error {
				if msg.Content != "test message" {
					t.Errorf("content = %q", msg.Content)
				}
				return nil
			},
		}
		var iface ChannelManager = cm
		err := iface.SendMessage(context.Background(), bus.OutboundMessage{Content: "test message"})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("SendMedia delegates correctly", func(t *testing.T) {
		cm := &mockChannelManager{
			sendMediaFn: func(ctx context.Context, msg bus.OutboundMediaMessage) error {
				if len(msg.Parts) != 1 {
					t.Errorf("parts length = %d", len(msg.Parts))
				}
				return nil
			},
		}
		var iface ChannelManager = cm
		err := iface.SendMedia(context.Background(), bus.OutboundMediaMessage{
			Parts: []bus.MediaPart{{Type: "video"}},
		})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("SendPlaceholder delegates correctly", func(t *testing.T) {
		cm := &mockChannelManager{
			sendPlaceholderFn: func(ctx context.Context, channel, chatID string) bool {
				if channel != "telegram" || chatID != "456" {
					t.Errorf("SendPlaceholder(%q, %q)", channel, chatID)
				}
				return true
			},
		}
		var iface ChannelManager = cm
		ok := iface.SendPlaceholder(context.Background(), "telegram", "456")
		if !ok {
			t.Error("SendPlaceholder returned false")
		}
	})

	t.Run("DismissToolFeedback delegates correctly", func(t *testing.T) {
		called := false
		cm := &mockChannelManager{
			dismissToolFeedbackFn: func(ctx context.Context, channel, chatID string, outboundCtx *bus.InboundContext) {
				called = true
				if outboundCtx == nil {
					t.Error("outboundCtx should not be nil")
				}
				if outboundCtx.ChatID != "789" {
					t.Errorf("ChatID = %q", outboundCtx.ChatID)
				}
			},
		}
		var iface ChannelManager = cm
		iface.DismissToolFeedback(context.Background(), "telegram", "789", &bus.InboundContext{ChatID: "789"})
		if !called {
			t.Error("DismissToolFeedback was not called")
		}
	})
}

// Compile-time interface satisfaction checks
var _ MessageBus = (*mockMessageBus)(nil)
var _ ChannelManager = (*mockChannelManager)(nil)
