package adapters

import (
	"context"
	"testing"

	"github.com/conglinyizhi/SylastraClaws/pkg/bus"
)

// NewMessageBus takes *bus.MessageBus. bus.NewMessageBus() has no external
// dependencies, so we can test with a real instance.

func TestNewMessageBusWithRealBus(t *testing.T) {
	realBus := bus.NewMessageBus()
	adapter := NewMessageBus(realBus)
	if adapter == nil {
		t.Fatal("NewMessageBus returned nil")
	}

	t.Run("PublishInbound and receive via InboundChan", func(t *testing.T) {
		ctx := context.Background()
		msg := bus.InboundMessage{Content: "hello", Channel: "telegram"}

		err := adapter.PublishInbound(ctx, msg)
		if err != nil {
			t.Fatalf("PublishInbound error: %v", err)
		}

		select {
		case received := <-adapter.InboundChan():
			if received.Content != "hello" {
				t.Errorf("received content = %q, want %q", received.Content, "hello")
			}
			if received.Channel != "telegram" {
				t.Errorf("received channel = %q", received.Channel)
			}
		default:
			t.Fatal("expected message on InboundChan, got none")
		}
	})

	t.Run("PublishOutbound succeeds", func(t *testing.T) {
		ctx := context.Background()
		msg := bus.OutboundMessage{Channel: "discord", Content: "response"}

		err := adapter.PublishOutbound(ctx, msg)
		if err != nil {
			t.Fatalf("PublishOutbound error: %v", err)
		}
	})

	t.Run("PublishOutboundMedia succeeds", func(t *testing.T) {
		ctx := context.Background()
		msg := bus.OutboundMediaMessage{
			Channel: "telegram",
			Parts:   []bus.MediaPart{{Type: "image", Ref: "media://abc"}},
		}

		err := adapter.PublishOutboundMedia(ctx, msg)
		if err != nil {
			t.Fatalf("PublishOutboundMedia error: %v", err)
		}
	})

	t.Run("InboundChan returns non-nil channel", func(t *testing.T) {
		ch := adapter.InboundChan()
		if ch == nil {
			t.Fatal("InboundChan returned nil")
		}
	})

	t.Run("PublishInbound with canceled context returns error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := adapter.PublishInbound(ctx, bus.InboundMessage{})
		if err == nil {
			t.Error("expected error with canceled context")
		}
	})

	realBus.Close()
}
