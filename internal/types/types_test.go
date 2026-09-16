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
