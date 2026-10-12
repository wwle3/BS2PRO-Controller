package types

import (
	"bytes"
	"testing"
)

func TestDefaultSmartStartStopIsDelayed(t *testing.T) {
	if got := GetDefaultConfig(false).SmartStartStop; got != "delayed" {
		t.Fatalf("default smart start stop %q", got)
	}
}

func TestRepairKeepsStoredSmartStartStopOff(t *testing.T) {
	cfg := GetDefaultConfig(false)
	cfg.SmartStartStop = "off"
	cfg.Repair()
	if cfg.SmartStartStop != "off" {
		t.Fatalf("stored off became %q", cfg.SmartStartStop)
	}
	cfg.SmartStartStop = ""
	cfg.Repair()
	if cfg.SmartStartStop != "delayed" {
		t.Fatalf("empty smart start stop became %q", cfg.SmartStartStop)
	}
}

func TestStartStopModeName(t *testing.T) {
	for raw, want := range map[uint8]string{0: "off", 1: "immediate", 2: "delayed"} {
		got, ok := StartStopModeName(raw)
		if !ok || got != want {
			t.Fatalf("raw %d => %q ok %t", raw, got, ok)
		}
	}
	if got, ok := StartStopModeName(3); ok || got != "" {
		t.Fatalf("raw 3 treated as %q", got)
	}
}

func legacyProCommand(gear byte, rpm int, sum byte) []byte {
	cmd := make([]byte, 23)
	cmd[0], cmd[1], cmd[2], cmd[3] = 0x5a, 0xa5, 0x26, 0x05
	cmd[4] = gear
	cmd[5] = byte(rpm)
	cmd[6] = byte(rpm >> 8)
	cmd[7] = sum
	return cmd
}

func TestProGearPresetsMatchLegacyBytes(t *testing.T) {
	want := map[string][]GearCommand{
		"静音": {
			{Name: "1挡低", Command: legacyProCommand(0, 1300, 0x44), RPM: 1300},
			{Name: "1挡中", Command: legacyProCommand(0, 1700, 0xd5), RPM: 1700},
			{Name: "1挡高", Command: legacyProCommand(0, 1900, 0x9e), RPM: 1900},
		},
		"标准": {
			{Name: "2挡低", Command: legacyProCommand(1, 2100, 0x68), RPM: 2100},
			{Name: "2挡中", Command: legacyProCommand(1, 2400, 0x95), RPM: 2400},
			{Name: "2挡高", Command: legacyProCommand(1, 2700, 0xc2), RPM: 2700},
		},
		"强劲": {
			{Name: "3挡低", Command: legacyProCommand(2, 2800, 0x27), RPM: 2800},
			{Name: "3挡中", Command: legacyProCommand(2, 3000, 0xf0), RPM: 3000},
			{Name: "3挡高", Command: legacyProCommand(2, 3300, 0x1d), RPM: 3300},
		},
		"超频": {
			{Name: "4挡低", Command: legacyProCommand(3, 3500, 0xe7), RPM: 3500},
			{Name: "4挡中", Command: legacyProCommand(3, 3700, 0xb0), RPM: 3700},
			{Name: "4挡高", Command: legacyProCommand(3, 4000, 0xdd), RPM: 4000},
		},
	}
	got := GearPresets("BS2PRO")
	if len(got) != len(want) {
		t.Fatalf("pro preset count %d", len(got))
	}
	for gear, commands := range want {
		actual := got[gear]
		if len(actual) != len(commands) {
			t.Fatalf("%s command count %d", gear, len(actual))
		}
		for i := range commands {
			if actual[i].Name != commands[i].Name || actual[i].RPM != commands[i].RPM || !bytes.Equal(actual[i].Command, commands[i].Command) {
				t.Fatalf("%s[%d] = %+v", gear, i, actual[i])
			}
			if !bytes.Equal(GearCommands[gear][i].Command, commands[i].Command) {
				t.Fatalf("GearCommands %s drifted", gear)
			}
		}
	}
}

func TestBS2StrongMidIs2900(t *testing.T) {
	commands := GearPresets("BS2")["强劲"]
	if len(commands) != 3 || commands[1].RPM != 2900 || commands[1].Name != "3挡中" {
		t.Fatalf("bs2 strong = %+v", commands)
	}
	if _, ok := GearPresets("bs2")["超频"]; ok {
		t.Fatal("plain BS2 still has overclock")
	}
	proMid := GearPresets("BS2PRO")["强劲"][1]
	if commands[1].RPM == proMid.RPM || bytes.Equal(commands[1].Command, proMid.Command) {
		t.Fatal("plain BS2 strong mid reused the pro command")
	}
}

func TestNormalizeCustomSpeedRPM(t *testing.T) {
	cases := []struct{ in, out int }{
		{4331, 4300},
		{4350, 4400},
		{500, 500},
		{4500, 4500},
		{449, 500},
		{4600, 4500},
		{0, 0},
		{2000, 2000},
	}
	for _, tc := range cases {
		if got := NormalizeCustomSpeedRPM(tc.in); got != tc.out {
			t.Fatalf("NormalizeCustomSpeedRPM(%d)=%d, want %d", tc.in, got, tc.out)
		}
	}
}

func TestRepairSnapsCustomSpeedRPM(t *testing.T) {
	cfg := GetDefaultConfig(false)
	cfg.CustomSpeedRPM = 4331
	cfg.Repair()
	if cfg.CustomSpeedRPM != 4300 {
		t.Fatalf("custom speed not snapped: %d", cfg.CustomSpeedRPM)
	}
}

func TestRepairKeepsWindowAndTheme(t *testing.T) {
	cfg := GetDefaultConfig(false)
	cfg.WindowWidth = 900
	cfg.WindowHeight = 641
	cfg.WindowMaximised = true
	cfg.WindowHasPosition = true
	cfg.WindowX = 12
	cfg.WindowY = -4
	cfg.ThemeFollowSystem = false
	cfg.ThemeMode = "dark"
	cfg.Repair()
	if cfg.WindowWidth != 900 || cfg.WindowHeight != 641 || !cfg.WindowMaximised || !cfg.WindowHasPosition || cfg.WindowX != 12 || cfg.WindowY != -4 {
		t.Fatalf("window pref changed: %+v", cfg)
	}
	if cfg.ThemeFollowSystem || cfg.ThemeMode != "dark" {
		t.Fatalf("theme pref changed: %+v", cfg)
	}
}

func TestRepairFillsEmptyWindowAndTheme(t *testing.T) {
	cfg := GetDefaultConfig(false)
	cfg.WindowWidth = 0
	cfg.WindowHeight = 0
	cfg.ThemeMode = ""
	cfg.Repair()
	if cfg.WindowWidth != DefaultWindowWidth || cfg.WindowHeight != DefaultWindowHeight || cfg.ThemeMode != "light" {
		t.Fatalf("defaults not filled: %+v", cfg)
	}
}

func TestMonitorRequired(t *testing.T) {
	cfg := GetDefaultConfig(false)
	cfg.TrayEnabled = false
	cfg.NotificationsEnabled = false
	cfg.Hotkeys.Enabled = false
	if cfg.MonitorRequired() {
		t.Fatal("monitor should not be required")
	}
	cfg.TrayEnabled = true
	if !cfg.MonitorRequired() {
		t.Fatal("tray should require monitor")
	}
}

func TestRepairDropsObsoleteProcessSwitchHotkeys(t *testing.T) {
	cfg := GetDefaultConfig(false)
	cfg.Hotkeys.Global = append(cfg.Hotkeys.Global, HotkeyBinding{
		Action: "toggle-process-switch", Accelerator: "Ctrl+Alt+P", Scope: "global", Enabled: true,
	})
	cfg.Hotkeys.InApp = append(cfg.Hotkeys.InApp, HotkeyBinding{
		Action: "navigate-process-switch", Accelerator: "Alt+5", Scope: "app", Enabled: true,
	})
	cfg.Repair()
	for _, binding := range append(append([]HotkeyBinding{}, cfg.Hotkeys.Global...), cfg.Hotkeys.InApp...) {
		if isObsoleteHotkeyAction(binding.Action) {
			t.Fatalf("obsolete hotkey kept: %s", binding.Action)
		}
	}
}

func TestDefaultHotkeysExcludeObsoleteActions(t *testing.T) {
	cfg := GetDefaultHotkeyConfig()
	for _, binding := range append(append([]HotkeyBinding{}, cfg.Global...), cfg.InApp...) {
		if isObsoleteHotkeyAction(binding.Action) {
			t.Fatalf("default still has %s", binding.Action)
		}
	}
}

func TestRepairDropsNavigateHotkeys(t *testing.T) {
	cfg := GetDefaultConfig(false)
	cfg.Hotkeys.InApp = append(cfg.Hotkeys.InApp, HotkeyBinding{
		Action: "navigate-dashboard", Accelerator: "Alt+1", Scope: "app", Enabled: true,
	})
	cfg.Repair()
	for _, binding := range cfg.Hotkeys.InApp {
		if isObsoleteHotkeyAction(binding.Action) {
			t.Fatalf("navigate hotkey kept: %s", binding.Action)
		}
	}
}

func TestRepairDropsInAppHotkeys(t *testing.T) {
	cfg := GetDefaultConfig(false)
	if len(cfg.Hotkeys.InApp) != 0 {
		t.Fatalf("default in-app hotkeys should be empty: %+v", cfg.Hotkeys.InApp)
	}
	cfg.Hotkeys.InApp = []HotkeyBinding{
		{Action: "save-context", Accelerator: "Ctrl+S", Scope: "app", Enabled: true},
		{Action: "escape-local-interaction", Accelerator: "Escape", Scope: "app", Enabled: true},
		{Action: "toggle-sidebar", Accelerator: "Ctrl+B", Scope: "app", Enabled: true},
	}
	cfg.Repair()
	if len(cfg.Hotkeys.InApp) != 0 {
		t.Fatalf("in-app hotkeys kept: %+v", cfg.Hotkeys.InApp)
	}
}
