package config

import (
	"testing"
)

func boolPtr(v bool) *bool           { return &v }
func intPtr(v int) *int              { return &v }
func stringPtr(v string) *string     { return &v }

func TestEffectiveShowReasoning(t *testing.T) {
	t.Run("nil behavior uses global default", func(t *testing.T) {
		var b *ChannelBehaviorConfig
		if got := b.EffectiveShowReasoning(true); got != true {
			t.Errorf("nil + global=true: got %v, want true", got)
		}
		if got := b.EffectiveShowReasoning(false); got != false {
			t.Errorf("nil + global=false: got %v, want false", got)
		}
	})

	t.Run("channel override takes precedence", func(t *testing.T) {
		b := &ChannelBehaviorConfig{ShowReasoning: boolPtr(true)}
		if got := b.EffectiveShowReasoning(false); got != true {
			t.Errorf("override=true + global=false: got %v, want true", got)
		}

		b2 := &ChannelBehaviorConfig{ShowReasoning: boolPtr(false)}
		if got := b2.EffectiveShowReasoning(true); got != false {
			t.Errorf("override=false + global=true: got %v, want false", got)
		}
	})

	t.Run("nil override falls back", func(t *testing.T) {
		b := &ChannelBehaviorConfig{}
		if got := b.EffectiveShowReasoning(true); got != true {
			t.Errorf("nil override + global=true: got %v, want true", got)
		}
	})
}

func TestEffectiveShowTokenFlow(t *testing.T) {
	t.Run("nil behavior uses global default", func(t *testing.T) {
		var b *ChannelBehaviorConfig
		if got := b.EffectiveShowTokenFlow(true); got != true {
			t.Errorf("nil + global=true: got %v, want true", got)
		}
	})

	t.Run("channel override takes precedence", func(t *testing.T) {
		b := &ChannelBehaviorConfig{ShowTokenFlow: boolPtr(false)}
		if got := b.EffectiveShowTokenFlow(true); got != false {
			t.Errorf("override=false + global=true: got %v, want false", got)
		}
	})
}

func TestEffectiveTokenFlowInterval(t *testing.T) {
	t.Run("nil behavior uses global default", func(t *testing.T) {
		var b *ChannelBehaviorConfig
		if got := b.EffectiveTokenFlowInterval(5); got != 5 {
			t.Errorf("nil + global=5: got %d, want 5", got)
		}
		if got := b.EffectiveTokenFlowInterval(0); got != 3 {
			t.Errorf("nil + global=0: got %d, want 3 (fallback)", got)
		}
	})

	t.Run("channel override takes precedence", func(t *testing.T) {
		b := &ChannelBehaviorConfig{TokenFlowIntervalSec: intPtr(10)}
		if got := b.EffectiveTokenFlowInterval(5); got != 10 {
			t.Errorf("override=10 + global=5: got %d, want 10", got)
		}
	})

	t.Run("invalid override (< 1) falls back", func(t *testing.T) {
		b := &ChannelBehaviorConfig{TokenFlowIntervalSec: intPtr(0)}
		if got := b.EffectiveTokenFlowInterval(5); got != 5 {
			t.Errorf("override=0 + global=5: got %d, want 5", got)
		}
		b2 := &ChannelBehaviorConfig{TokenFlowIntervalSec: intPtr(-1)}
		if got := b2.EffectiveTokenFlowInterval(5); got != 5 {
			t.Errorf("override=-1 + global=5: got %d, want 5", got)
		}
	})
}

func TestEffectiveThinkingOverride(t *testing.T) {
	t.Run("nil behavior returns empty", func(t *testing.T) {
		var b *ChannelBehaviorConfig
		if got := b.EffectiveThinkingOverride(); got != "" {
			t.Errorf("nil: got %q, want empty", got)
		}
	})

	t.Run("empty override returns empty", func(t *testing.T) {
		b := &ChannelBehaviorConfig{}
		if got := b.EffectiveThinkingOverride(); got != "" {
			t.Errorf("empty: got %q, want empty", got)
		}
	})

	t.Run("override returns value", func(t *testing.T) {
		b := &ChannelBehaviorConfig{ThinkingOverride: "high"}
		if got := b.EffectiveThinkingOverride(); got != "high" {
			t.Errorf("override=high: got %q, want high", got)
		}
	})
}

func TestChannelBehaviorInChannel(t *testing.T) {
	t.Run("channel without behavior", func(t *testing.T) {
		ch := &Channel{}
		if ch.Behavior != nil {
			t.Error("new Channel should have nil Behavior")
		}
	})

	t.Run("channel with behavior", func(t *testing.T) {
		ch := &Channel{
			Behavior: &ChannelBehaviorConfig{
				ShowReasoning: boolPtr(false),
			},
		}
		if ch.Behavior == nil {
			t.Fatal("Behavior should not be nil")
		}
		if got := ch.Behavior.EffectiveShowReasoning(true); got != false {
			t.Errorf("want override=false, got %v", got)
		}
	})
}

func TestAgentDefaultsBehaviorDefaults(t *testing.T) {
	t.Run("defaults nil behavior", func(t *testing.T) {
		d := &AgentDefaults{}
		if d.BehaviorDefaults != nil {
			t.Error("new AgentDefaults should have nil BehaviorDefaults")
		}
	})

	t.Run("defaults with behavior", func(t *testing.T) {
		d := &AgentDefaults{
			BehaviorDefaults: &ChannelBehaviorConfig{
				ShowReasoning: boolPtr(true),
				ShowTokenFlow: boolPtr(false),
			},
		}
		if got := d.BehaviorDefaults.EffectiveShowReasoning(false); got != true {
			t.Errorf("want true, got %v", got)
		}
		if got := d.BehaviorDefaults.EffectiveShowTokenFlow(true); got != false {
			t.Errorf("want false, got %v", got)
		}
	})
}
