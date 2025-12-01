package device

import (
	"testing"
)

func TestSpeed_String(t *testing.T) {
	tests := []struct {
		speed Speed
		want  string
	}{
		{SpeedLow, "Low Speed (1.5 Mbps)"},
		{SpeedFull, "Full Speed (12 Mbps)"},
		{SpeedHigh, "High Speed (480 Mbps)"},
		{SpeedSuper, "Super Speed (5 Gbps)"},
		{Speed(99), "Unknown Speed (99)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.speed.String(); got != tt.want {
				t.Errorf("Speed.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSpeed_MaxPacketSize0(t *testing.T) {
	tests := []struct {
		speed Speed
		want  uint16
	}{
		{SpeedLow, 8},
		{SpeedFull, 64},
		{SpeedHigh, 64},
		{SpeedSuper, 512},
		{Speed(99), 8},
	}

	for _, tt := range tests {
		t.Run(tt.speed.String(), func(t *testing.T) {
			if got := tt.speed.MaxPacketSize0(); got != tt.want {
				t.Errorf("Speed.MaxPacketSize0() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestState_String(t *testing.T) {
	tests := []struct {
		state State
		want  string
	}{
		{StateAttached, "Attached"},
		{StatePowered, "Powered"},
		{StateDefault, "Default"},
		{StateAddress, "Address"},
		{StateConfigured, "Configured"},
		{StateSuspended, "Suspended"},
		{State(99), "Unknown State (99)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.state.String(); got != tt.want {
				t.Errorf("State.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
