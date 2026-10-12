package device

import "testing"

func TestParseFanDataReadsVbusAndFlags(t *testing.T) {
	data := []byte{
		0x00,
		0x5A, 0xA5,
		0xEF,
		0x00,
		0x60,
		0x0D,
		0x00,
		0xE8, 0x03,
		0xD0, 0x07,
	}
	got := (&Manager{}).parseFanData(data, len(data))
	if got == nil {
		t.Fatal("expected fan data")
	}
	if got.Vbus != 3 {
		t.Fatalf("vbus = %d, want 3", got.Vbus)
	}
	if !got.SmartFreq || got.PowerOnAutoStart || got.IntelligentStartStop != 3 {
		t.Fatalf("flags smart=%t power=%t start=%d", got.SmartFreq, got.PowerOnAutoStart, got.IntelligentStartStop)
	}
	if got.CurrentRPM != 1000 || got.TargetRPM != 2000 {
		t.Fatalf("rpm current=%d target=%d", got.CurrentRPM, got.TargetRPM)
	}
}

func TestParseFanDataRejectsWrongHeader(t *testing.T) {
	data := []byte{0x00, 0x5A, 0xA5, 0xEE, 0, 0, 0, 0, 0, 0, 0}
	if got := (&Manager{}).parseFanData(data, len(data)); got != nil {
		t.Fatal("expected reject")
	}
}
