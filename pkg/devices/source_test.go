package devices

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/devices/events"
)

// TestEventSourceTypeAlias verifies that EventSource aliases events.EventSource
// and can be used interchangeably.
func TestEventSourceTypeAlias(t *testing.T) {
	// Compile-time check: EventSource is assignable from events.EventSource
	var _ events.EventSource = (EventSource)(nil)
	var _ EventSource = (events.EventSource)(nil)

	// Verify that a nil EventSource of either type is nil
	t.Run("nil values are nil", func(t *testing.T) {
		var es EventSource
		if es != nil {
			t.Error("EventSource should be nil by default")
		}
	})

	t.Run("type identity", func(t *testing.T) {
		var es1 events.EventSource
		var es2 EventSource = es1
		var es3 events.EventSource = es2
		_ = es3
		// These assignments compile, confirming type identity.
	})
}
