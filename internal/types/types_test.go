package types

import "testing"

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

func TestDefaultHotkeysExcludeProcessSwitch(t *testing.T) {
	cfg := GetDefaultHotkeyConfig()
	for _, binding := range append(append([]HotkeyBinding{}, cfg.Global...), cfg.InApp...) {
		if isObsoleteHotkeyAction(binding.Action) {
			t.Fatalf("default still has %s", binding.Action)
		}
	}
}
