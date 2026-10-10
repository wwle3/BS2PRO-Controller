package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/TIANLI0/BS2PRO-Controller/internal/config"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	windowDefaultWidth  = 900
	windowDefaultHeight = 620
)

type windowSizePreference struct {
	Width     int  `json:"width"`
	Height    int  `json:"height"`
	Maximised bool `json:"maximised"`
}

func defaultWindowSize() windowSizePreference {
	return windowSizePreference{Width: windowDefaultWidth, Height: windowDefaultHeight}
}

func windowSizePath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return filepath.Join(config.GetInstallDir(), "window-size.json")
	}
	return filepath.Join(configDir, "BS2PRO-Controller", "window-size.json")
}

func acceptedWindowSize(width, height int) (int, int, bool) {
	if width < windowDefaultWidth || height < windowDefaultHeight {
		return 0, 0, false
	}
	return width, height, true
}

func loadWindowSize() windowSizePreference {
	data, err := os.ReadFile(windowSizePath())
	if err != nil {
		return defaultWindowSize()
	}
	pref := defaultWindowSize()
	if err := json.Unmarshal(data, &pref); err != nil {
		return defaultWindowSize()
	}
	if _, _, ok := acceptedWindowSize(pref.Width, pref.Height); !ok {
		return defaultWindowSize()
	}
	return pref
}

func saveWindowSize(ctx context.Context) {
	if ctx == nil || runtime.WindowIsMinimised(ctx) || runtime.WindowIsFullscreen(ctx) {
		return
	}
	next := loadWindowSize()
	if runtime.WindowIsMaximised(ctx) {
		next.Maximised = true
	} else {
		width, height := runtime.WindowGetSize(ctx)
		width, height, ok := acceptedWindowSize(width, height)
		if !ok {
			return
		}
		next.Width = width
		next.Height = height
		next.Maximised = false
	}
	path := windowSizePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	data, err := json.Marshal(next)
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o644)
}
