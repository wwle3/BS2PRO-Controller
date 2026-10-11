package main

import (
	"context"
	"encoding/json"
	"os"
	"unsafe"

	"github.com/TIANLI0/BS2PRO-Controller/internal/config"
	"github.com/TIANLI0/BS2PRO-Controller/internal/types"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows"
)

const (
	windowDefaultWidth  = types.DefaultWindowWidth
	windowDefaultHeight = types.DefaultWindowHeight
)

type windowSizePreference struct {
	Width       int
	Height      int
	Maximised   bool
	X           int
	Y           int
	HasPosition bool
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

func acceptedWindowSize(width, height int) (int, int, bool) {
	if width < windowDefaultWidth || height < windowDefaultHeight {
		return 0, 0, false
	}
	return width, height, true
}

func readStoredConfig() types.AppConfig {
	cfg := types.GetDefaultConfig(false)
	data, err := os.ReadFile(config.DefaultConfigPath(config.GetInstallDir()))
	if err != nil {
		return cfg
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return types.GetDefaultConfig(false)
	}
	return cfg
}

func loadWindowSize() windowSizePreference {
	cfg := readStoredConfig()
	pref := windowSizePreference{
		Width: cfg.WindowWidth, Height: cfg.WindowHeight, Maximised: cfg.WindowMaximised,
		X: cfg.WindowX, Y: cfg.WindowY, HasPosition: cfg.WindowHasPosition,
	}
	if _, _, ok := acceptedWindowSize(pref.Width, pref.Height); !ok {
		return defaultWindowSize()
	}
	return pref
}

func (a *App) saveWindowSize(ctx context.Context) {
	if ctx == nil || runtime.WindowIsMinimised(ctx) || runtime.WindowIsFullscreen(ctx) {
		return
	}
	maximised := runtime.WindowIsMaximised(ctx)
	width, height, ok := 0, 0, false
	x, y := 0, 0
	if !maximised {
		width, height = runtime.WindowGetSize(ctx)
		width, height, ok = acceptedWindowSize(width, height)
		if !ok {
			return
		}
		x, y = runtime.WindowGetPosition(ctx)
	}
	_ = a.persistConfigPatch(func(cfg *types.AppConfig) {
		if maximised {
			cfg.WindowMaximised = true
			return
		}
		cfg.WindowWidth = width
		cfg.WindowHeight = height
		cfg.WindowMaximised = false
		cfg.WindowX = x
		cfg.WindowY = y
		cfg.WindowHasPosition = true
	})
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
