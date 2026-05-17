package events

import (
	"strings"
	"testing"
)

func TestDeviceEventFormatMessage(t *testing.T) {
	tests := []struct {
		name string
		evt  DeviceEvent
		want []string // substrings that should be present
	}{
		{
			name: "add action",
			evt: DeviceEvent{
				Action:       ActionAdd,
				Kind:         KindUSB,
				DeviceID:     "1-2",
				Vendor:       "Sipeed",
				Product:      "SylastraClaws",
				Serial:       "SN12345",
				Capabilities: "Audio, Serial",
			},
			want: []string{"🔌", "Device", "Connected", "Type: usb", "Device: Sipeed SylastraClaws", "Capabilities: Audio, Serial", "Serial: SN12345"},
		},
		{
			name: "remove action",
			evt: DeviceEvent{
				Action:  ActionRemove,
				Kind:    KindBluetooth,
				Vendor:  "Generic",
				Product: "BT Device",
			},
			want: []string{"🔌", "Device", "Disconnected", "Type: bluetooth", "Device: Generic BT Device"},
		},
		{
			name: "change action",
			evt: DeviceEvent{
				Action:  ActionChange,
				Kind:    KindPCI,
				Vendor:  "Intel",
				Product: "WiFi Card",
			},
			want: []string{"🔌", "Device", "Connected", "Type: pci", "Device: Intel WiFi Card"},
		},
		{
			name: "no capabilities or serial",
			evt: DeviceEvent{
				Action:  ActionAdd,
				Kind:    KindGeneric,
				Vendor:  "Unknown",
				Product: "Device",
			},
			want: []string{"🔌", "Connected", "Type: generic", "Device: Unknown Device"},
		},
		{
			name: "with serial but no capabilities",
			evt: DeviceEvent{
				Action: ActionAdd,
				Kind:   KindUSB,
				Vendor: "Foo",
				Product: "Bar",
				Serial: "SERIAL001",
			},
			want: []string{"Serial: SERIAL001"},
		},
		{
			name: "with capabilities but no serial",
			evt: DeviceEvent{
				Action:       ActionAdd,
				Kind:         KindUSB,
				Vendor:       "Vendor",
				Product:      "Product",
				Capabilities: "HID",
			},
			want: []string{"Capabilities: HID"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.evt.FormatMessage()
			for _, substr := range tt.want {
				if !strings.Contains(msg, substr) {
					t.Errorf("FormatMessage() missing expected substring %q\nFull output:\n%s", substr, msg)
				}
			}
		})
	}
}

func TestDeviceEventFormatMessageNotContains(t *testing.T) {
	t.Run("no serial when serial is empty", func(t *testing.T) {
		evt := DeviceEvent{
			Action:  ActionAdd,
			Kind:    KindUSB,
			Vendor:  "Test",
			Product: "Device",
		}
		msg := evt.FormatMessage()
		if strings.Contains(msg, "Serial:") {
			t.Error("FormatMessage should not include Serial: when empty")
		}
	})

	t.Run("no capabilities when empty", func(t *testing.T) {
		evt := DeviceEvent{
			Action:  ActionAdd,
			Kind:    KindUSB,
			Vendor:  "Test",
			Product: "Device",
		}
		msg := evt.FormatMessage()
		if strings.Contains(msg, "Capabilities:") {
			t.Error("FormatMessage should not include Capabilities: when empty")
		}
	})
}

func TestFormatMessageNotEmpty(t *testing.T) {
	// Ensure the message is never empty for valid input
	evt := DeviceEvent{
		Action:  ActionAdd,
		Kind:    KindUSB,
		Vendor:  "Test",
		Product: "Device",
	}
	msg := evt.FormatMessage()
	if len(msg) == 0 {
		t.Error("FormatMessage() should not return empty string")
	}
}

func TestActionKinds(t *testing.T) {
	if ActionAdd != "add" {
		t.Errorf("ActionAdd = %q, want \"add\"", ActionAdd)
	}
	if ActionRemove != "remove" {
		t.Errorf("ActionRemove = %q, want \"remove\"", ActionRemove)
	}
	if ActionChange != "change" {
		t.Errorf("ActionChange = %q, want \"change\"", ActionChange)
	}
	if KindUSB != "usb" {
		t.Errorf("KindUSB = %q, want \"usb\"", KindUSB)
	}
	if KindBluetooth != "bluetooth" {
		t.Errorf("KindBluetooth = %q, want \"bluetooth\"", KindBluetooth)
	}
	if KindPCI != "pci" {
		t.Errorf("KindPCI = %q, want \"pci\"", KindPCI)
	}
	if KindGeneric != "generic" {
		t.Errorf("KindGeneric = %q, want \"generic\"", KindGeneric)
	}
}
