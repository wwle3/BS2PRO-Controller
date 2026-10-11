package main

import "github.com/TIANLI0/BS2PRO-Controller/internal/types"

type ThemePreference struct {
	FollowSystem bool   `json:"followSystem"`
	Mode         string `json:"mode"`
}

func defaultThemePreference() ThemePreference {
	return ThemePreference{FollowSystem: true, Mode: "light"}
}

func loadThemePreference() ThemePreference {
	cfg := readStoredConfig()
	if cfg.ThemeMode != "dark" && cfg.ThemeMode != "light" {
		return defaultThemePreference()
	}
	return ThemePreference{FollowSystem: cfg.ThemeFollowSystem, Mode: cfg.ThemeMode}
}

func (a *App) saveThemePreference(pref ThemePreference) error {
	if pref.Mode != "dark" && pref.Mode != "light" {
		pref.Mode = "light"
	}
	return a.persistConfigPatch(func(cfg *types.AppConfig) {
		cfg.ThemeFollowSystem = pref.FollowSystem
		cfg.ThemeMode = pref.Mode
	})
}
