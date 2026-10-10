package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"unsafe"

	"github.com/TIANLI0/BS2PRO-Controller/internal/config"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows"
)

const (
	windowDefaultWidth  = 900
	windowDefaultHeight = 620
)

type windowSizePreference struct {
	Width       int  `json:"width"`
	Height      int  `json:"height"`
	Maximised   bool `json:"maximised"`
	X           int  `json:"x"`
	Y           int  `json:"y"`
	HasPosition bool `json:"hasPosition"`
}

var (
	user32              = windows.NewLazySystemDLL("user32.dll")
	procMonitorFromRect = user32.NewProc("MonitorFromRect")
	procGetMonitorInfoW = user32.NewProc("GetMonitorInfoW")
)

type winRect struct {
	Left, Top, Right, Bottom int32
}

type winMonitorInfo struct {
	CbSize    uint32
	RcMonitor winRect
	RcWork    winRect
	DwFlags   uint32
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
		x, y := runtime.WindowGetPosition(ctx)
		next.X = x
		next.Y = y
		next.HasPosition = true
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

func restoreWindowPosition(ctx context.Context) {
	if ctx == nil {
		return
	}
	pref := loadWindowSize()
	if pref.Maximised || !pref.HasPosition {
		return
	}
	if _, _, onScreen := monitorWorkOrigin(pref.X, pref.Y); !onScreen {
		return
	}
	curX, curY := runtime.WindowGetPosition(ctx)
	originX, originY, ok := monitorWorkOrigin(curX, curY)
	if !ok {
		return
	}
	x, y := windowPositionForSet(pref.X, pref.Y, originX, originY)
	runtime.WindowSetPosition(ctx, x, y)
}

// Wails v2.16 的 WindowGetPosition 返回屏幕坐标，WindowSetPosition 却从当前显示器工作区原点加偏移。
func windowPositionForSet(savedX, savedY, originX, originY int) (int, int) {
	return savedX - originX, savedY - originY
}

func monitorWorkOrigin(x, y int) (int, int, bool) {
	rect := winRect{Left: int32(x), Top: int32(y), Right: int32(x) + 1, Bottom: int32(y) + 1}
	handle, _, _ := procMonitorFromRect.Call(uintptr(unsafe.Pointer(&rect)), 0)
	if handle == 0 {
		return 0, 0, false
	}
	info := winMonitorInfo{CbSize: uint32(unsafe.Sizeof(winMonitorInfo{}))}
	ok, _, _ := procGetMonitorInfoW.Call(handle, uintptr(unsafe.Pointer(&info)))
	if ok == 0 {
		return 0, 0, false
	}
	return int(info.RcWork.Left), int(info.RcWork.Top), true
}
