// Package types 定义了 BS2PRO 控制器应用中使用的所有共享类型
package types

import (
	"sort"
	"strings"
)

// FanCurvePoint 风扇曲线点
type FanCurvePoint struct {
	Temperature int `json:"temperature"` // 温度 °C
	RPM         int `json:"rpm"`         // 转速 RPM
	Offset      int `json:"offset"`      // 转速偏移 RPM (100的整数倍)
}

// FanData 风扇数据结构
type FanData struct {
	ReportID             uint8  `json:"reportId"`
	MagicSync            uint16 `json:"magicSync"`
	Command              uint8  `json:"command"`
	Status               uint8  `json:"status"`
	GearSettings         uint8  `json:"gearSettings"`
	CurrentMode          uint8  `json:"currentMode"`
	Reserved1            uint8  `json:"reserved1"`
	CurrentRPM           uint16 `json:"currentRpm"`
	TargetRPM            uint16 `json:"targetRpm"`
	Vbus                 uint8  `json:"vbus"`
	SmartFreq            bool   `json:"smartFreq"`
	PowerOnAutoStart     bool   `json:"powerOnAutoStart"`
	IntelligentStartStop uint8  `json:"intelligentStartStop"`
}

// StartStopModeName 把心跳 data[6] 的 bit2-3 译成启停档。3 以及其他值不是档。
func StartStopModeName(raw uint8) (string, bool) {
	switch raw {
	case 0:
		return "off", true
	case 1:
		return "immediate", true
	case 2:
		return "delayed", true
	default:
		return "", false
	}
}

// GearCommand 挡位命令结构
type GearCommand struct {
	Name    string `json:"name"`    // 挡位名称
	Command []byte `json:"command"` // 命令字节
	RPM     int    `json:"rpm"`     // 对应转速
}

// TemperatureData 温度数据
type TemperatureData struct {
	CPUTemp     int    `json:"cpuTemp"`       // CPU温度
	GPUTemp     int    `json:"gpuTemp"`       // GPU温度
	MaxTemp     int    `json:"maxTemp"`       // 最高温度
	UpdateTime  int64  `json:"updateTime"`    // 更新时间戳
	BridgeOk    bool   `json:"bridgeOk"`      // 桥接程序是否正常
	BridgeMsg   string `json:"bridgeMessage"` // 桥接故障提示
	AutoOffset  int    `json:"autoOffset"`    // 自动偏移量 (由 fanoffset 控制器计算)
	EngineState string `json:"engineState"`   // 当前温度区间引擎状态（补偿中/回收中/稳定）
}

// BridgeTemperatureData 桥接程序返回的温度数据
type BridgeTemperatureData struct {
	CpuTemp    int    `json:"cpuTemp"`
	GpuTemp    int    `json:"gpuTemp"`
	MaxTemp    int    `json:"maxTemp"`
	UpdateTime int64  `json:"updateTime"`
	Success    bool   `json:"success"`
	Error      string `json:"error"`
}

// RGBColorConfig RGB颜色配置
type RGBColorConfig struct {
	R int `json:"r"`
	G int `json:"g"`
	B int `json:"b"`
}

// RGBConfig RGB灯效配置
type RGBConfig struct {
	Mode       string           `json:"mode"`
	Colors     []RGBColorConfig `json:"colors"`
	Speed      string           `json:"speed"`
	Brightness int              `json:"brightness"`
}

// AppConfig 应用配置
type AppConfig struct {
	AutoControl             bool             `json:"autoControl"`             // 智能变频开关
	FanCurve                []FanCurvePoint  `json:"fanCurve"`                // 风扇曲线
	GearLight               bool             `json:"gearLight"`               // 挡位灯
	NotificationsEnabled    bool             `json:"notificationsEnabled"`    // 系统通知开关
	TrayEnabled             bool             `json:"trayEnabled"`             // 托盘菜单开关
	Hotkeys                 *HotkeyConfig    `json:"hotkeys"`                 // 快捷键配置
	PowerOnStart            bool             `json:"powerOnStart"`            // 通电自启
	WindowsAutoStart        bool             `json:"windowsAutoStart"`        // Windows开机自启动
	MonitorAutoStart        bool             `json:"monitorAutoStart"`        // Monitor开机自启动
	SmartStartStop          string           `json:"smartStartStop"`          // 智能启停
	Brightness              int              `json:"brightness"`              // 亮度
	TempUpdateRate          int              `json:"tempUpdateRate"`          // 温度更新频率(秒)
	TempSampleCount         int              `json:"tempSampleCount"`         // 温度采样次数(用于平均)
	ConfigPath              string           `json:"configPath"`              // 配置文件路径
	ManualGear              string           `json:"manualGear"`              // 手动挡位设置
	ManualLevel             string           `json:"manualLevel"`             // 手动挡位级别(低中高)
	DebugMode               bool             `json:"debugMode"`               // 调试模式
	GuiMonitoring           bool             `json:"guiMonitoring"`           // GUI监控开关
	CustomSpeedEnabled      bool             `json:"customSpeedEnabled"`      // 自定义转速开关
	CustomSpeedRPM          int              `json:"customSpeedRPM"`          // 自定义转速，500–4500 且按 100 取整
	IgnoreDeviceOnReconnect bool             `json:"ignoreDeviceOnReconnect"` // 断连后忽略设备状态(保持APP配置)
	FanCurveOffsetEnabled   bool             `json:"fanCurveOffsetEnabled"`   // 风扇曲线偏移开关
	RGBConfig               *RGBConfig       `json:"rgbConfig"`               // RGB灯效配置
	HotkeyConflicts         []HotkeyConflict `json:"hotkeyConflicts,omitempty"`
	WindowWidth             int              `json:"windowWidth"`       // 主窗口宽度
	WindowHeight            int              `json:"windowHeight"`      // 主窗口高度
	WindowMaximised         bool             `json:"windowMaximised"`   // 主窗口是否最大化
	WindowX                 int              `json:"windowX"`           // 主窗口屏幕坐标 X
	WindowY                 int              `json:"windowY"`           // 主窗口屏幕坐标 Y
	WindowHasPosition       bool             `json:"windowHasPosition"` // 是否已记住位置
	ThemeFollowSystem       bool             `json:"themeFollowSystem"` // 主题跟随系统
	ThemeMode               string           `json:"themeMode"`         // 主题 light 或 dark
}

const (
	DefaultWindowWidth  = 900
	DefaultWindowHeight = 620
	CustomSpeedMinRPM   = 500
	CustomSpeedMaxRPM   = 4500
	CustomSpeedStepRPM  = 100
)

// NormalizeCustomSpeedRPM 把自定义转速收到正常档位。非正数原样返回。
func NormalizeCustomSpeedRPM(rpm int) int {
	if rpm <= 0 {
		return rpm
	}
	rpm = ((rpm + CustomSpeedStepRPM/2) / CustomSpeedStepRPM) * CustomSpeedStepRPM
	if rpm < CustomSpeedMinRPM {
		return CustomSpeedMinRPM
	}
	if rpm > CustomSpeedMaxRPM {
		return CustomSpeedMaxRPM
	}
	return rpm
}

type HotkeyConfig struct {
	Enabled bool            `json:"enabled"`
	Global  []HotkeyBinding `json:"global"`
	InApp   []HotkeyBinding `json:"inApp"`
}

type HotkeyBinding struct {
	Action      string `json:"action"`
	Accelerator string `json:"accelerator"`
	Scope       string `json:"scope"`
	Enabled     bool   `json:"enabled"`
	Editable    bool   `json:"editable"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

type HotkeyConflict struct {
	Accelerator string   `json:"accelerator"`
	Scopes      []string `json:"scopes"`
	Actions     []string `json:"actions"`
	Source      string   `json:"source,omitempty"`
	Message     string   `json:"message,omitempty"`
}

// Logger 日志记录器接口
type Logger interface {
	// 长期接口保持 message + key/value 形态。
	// 具体实现可在迁移期兼容旧 printf 调用，但业务侧不再暴露单独的 *f 方法。
	// message 语义上应传 string，这里保留 any 以避免 go vet 将其视为 printf 包装。
	Info(msg any, args ...any)
	Error(msg any, args ...any)
	Warn(msg any, args ...any)
	Debug(msg any, args ...any)
	Close()
	CleanOldLogs()
	SetDebugMode(enabled bool)
	GetLogDir() string
}

func makeGearCommand(name string, gear byte, rpm int) GearCommand {
	lo := byte(rpm)
	hi := byte(rpm >> 8)
	sum := byte(0x26 + 0x05 + int(gear) + int(lo) + int(hi))
	cmd := make([]byte, 23)
	copy(cmd, []byte{0x5a, 0xa5, 0x26, 0x05, gear, lo, hi, sum})
	return GearCommand{Name: name, Command: cmd, RPM: rpm}
}

func gearTrio(prefix string, gear byte, low, mid, high int) []GearCommand {
	return []GearCommand{
		makeGearCommand(prefix+"低", gear, low),
		makeGearCommand(prefix+"中", gear, mid),
		makeGearCommand(prefix+"高", gear, high),
	}
}

func proGearPresets() map[string][]GearCommand {
	return map[string][]GearCommand{
		"静音": gearTrio("1挡", 0, 1300, 1700, 1900),
		"标准": gearTrio("2挡", 1, 2100, 2400, 2700),
		"强劲": gearTrio("3挡", 2, 2800, 3000, 3300),
		"超频": gearTrio("4挡", 3, 3500, 3700, 4000),
	}
}

func bs2GearPresets() map[string][]GearCommand {
	return map[string][]GearCommand{
		"静音": gearTrio("1挡", 0, 1300, 1700, 1900),
		"标准": gearTrio("2挡", 1, 2100, 2400, 2700),
		"强劲": gearTrio("3挡", 2, 2800, 2900, 3000),
	}
}

// IsPlainBS2 判断是不是 BS2
func IsPlainBS2(model string) bool {
	compact := strings.ToUpper(strings.ReplaceAll(model, " ", ""))
	return compact == "BS2"
}

// GearPresets 按机型返回预设
func GearPresets(model string) map[string][]GearCommand {
	if IsPlainBS2(model) {
		return bs2GearPresets()
	}
	return proGearPresets()
}

// GearCommands 是 BS2 Pro 的预设，预设占位
var GearCommands = proGearPresets()

// GetDefaultFanCurve 获取默认风扇曲线
func GetDefaultFanCurve() []FanCurvePoint {
	return []FanCurvePoint{
		{Temperature: 30, RPM: 500, Offset: 0},
		{Temperature: 35, RPM: 1200, Offset: 0},
		{Temperature: 40, RPM: 1400, Offset: 0},
		{Temperature: 45, RPM: 1600, Offset: 0},
		{Temperature: 50, RPM: 1800, Offset: 100},
		{Temperature: 55, RPM: 2000, Offset: 100},
		{Temperature: 60, RPM: 2300, Offset: 100},
		{Temperature: 65, RPM: 2600, Offset: 200},
		{Temperature: 70, RPM: 2900, Offset: 200},
		{Temperature: 75, RPM: 3200, Offset: 200},
		{Temperature: 80, RPM: 3500, Offset: 300},
		{Temperature: 85, RPM: 3800, Offset: 200},
		{Temperature: 90, RPM: 4000, Offset: 0},
		{Temperature: 95, RPM: 4000, Offset: 0},
		{Temperature: 100, RPM: 4000, Offset: 0},
	}
}

// GetDefaultConfig 获取默认配置
func GetDefaultConfig(isAutoStart bool) AppConfig {
	cfg := AppConfig{
		AutoControl:             false,
		FanCurve:                GetDefaultFanCurve(),
		GearLight:               true,
		NotificationsEnabled:    true,
		TrayEnabled:             true,
		Hotkeys:                 GetDefaultHotkeyConfig(),
		PowerOnStart:            false,
		WindowsAutoStart:        false,
		MonitorAutoStart:        false,
		SmartStartStop:          "delayed",
		Brightness:              100,
		TempUpdateRate:          2,
		TempSampleCount:         1,
		ConfigPath:              "",
		ManualGear:              "标准",
		ManualLevel:             "中",
		DebugMode:               false,
		GuiMonitoring:           true,
		CustomSpeedEnabled:      false,
		CustomSpeedRPM:          2000,
		IgnoreDeviceOnReconnect: true, // 默认开启，防止断连后误判用户手动切换
		FanCurveOffsetEnabled:   false,
		WindowWidth:             DefaultWindowWidth,
		WindowHeight:            DefaultWindowHeight,
		ThemeFollowSystem:       true,
		ThemeMode:               "light",
		RGBConfig: &RGBConfig{
			Mode:       "smart",
			Colors:     []RGBColorConfig{{R: 0, G: 0, B: 255}, {R: 255, G: 0, B: 0}, {R: 0, G: 255, B: 0}},
			Speed:      "medium",
			Brightness: 100,
		},
	}
	cfg.NormalizeMonitorAutoStart()
	return cfg
}

func (c AppConfig) HotkeysRequireMonitor() bool {
	return c.Hotkeys != nil && c.Hotkeys.Enabled
}

func (c AppConfig) MonitorRequired() bool {
	return c.TrayEnabled || c.NotificationsEnabled || c.HotkeysRequireMonitor()
}

func (c *AppConfig) NormalizeMonitorAutoStart() bool {
	desired := c.MonitorRequired()
	if c.MonitorAutoStart == desired {
		return false
	}
	c.MonitorAutoStart = desired
	return true
}

func GetDefaultHotkeyConfig() *HotkeyConfig {
	return &HotkeyConfig{
		Enabled: true,
		Global: []HotkeyBinding{
			{Action: "show-main-window", Accelerator: "Ctrl+Alt+B", Scope: "global", Enabled: true, Editable: true, Description: "显示或隐藏主窗口", Category: "窗口"},
			{Action: "toggle-auto-control", Accelerator: "Ctrl+Alt+A", Scope: "global", Enabled: true, Editable: true, Description: "切换智能变频", Category: "设备"},
			{Action: "cycle-rgb-mode", Accelerator: "Ctrl+Alt+R", Scope: "global", Enabled: true, Editable: true, Description: "切换 RGB 灯光模式", Category: "RGB"},
		},
		InApp: []HotkeyBinding{},
	}
}

func NormalizeAccelerator(accelerator string) string {
	parts := strings.Split(strings.TrimSpace(accelerator), "+")
	mods := map[string]bool{}
	key := ""
	for _, part := range parts {
		token := strings.TrimSpace(strings.ToLower(part))
		switch token {
		case "control", "ctrl":
			mods["Ctrl"] = true
		case "alternate", "alt":
			mods["Alt"] = true
		case "shift":
			mods["Shift"] = true
		case "meta", "cmd", "win", "super":
			mods["Meta"] = true
		default:
			if token == "escape" {
				key = "Escape"
			} else {
				key = strings.ToUpper(token)
			}
		}
	}
	ordered := make([]string, 0, 5)
	for _, mod := range []string{"Ctrl", "Alt", "Shift", "Meta"} {
		if mods[mod] {
			ordered = append(ordered, mod)
		}
	}
	if key != "" {
		ordered = append(ordered, key)
	}
	return strings.Join(ordered, "+")
}

func DetectHotkeyConflicts(cfg *HotkeyConfig) []HotkeyConflict {
	if cfg == nil {
		return nil
	}
	type item struct{ scope, action string }
	index := map[string][]item{}
	for _, binding := range append(append([]HotkeyBinding{}, cfg.Global...), cfg.InApp...) {
		if !binding.Enabled {
			continue
		}
		accelerator := NormalizeAccelerator(binding.Accelerator)
		if accelerator == "" {
			continue
		}
		index[accelerator] = append(index[accelerator], item{scope: binding.Scope, action: binding.Action})
	}
	conflicts := make([]HotkeyConflict, 0)
	for accelerator, items := range index {
		groups := map[string]map[string]bool{}
		for _, item := range items {
			if groups[item.scope] == nil {
				groups[item.scope] = map[string]bool{}
			}
			groups[item.scope][item.action] = true
		}
		for scope, actions := range groups {
			if len(actions) < 2 {
				continue
			}
			list := make([]string, 0, len(actions))
			for action := range actions {
				list = append(list, action)
			}
			sort.Strings(list)
			conflicts = append(conflicts, HotkeyConflict{Accelerator: accelerator, Scopes: []string{scope}, Actions: list})
		}
	}
	sort.Slice(conflicts, func(i, j int) bool { return conflicts[i].Accelerator < conflicts[j].Accelerator })
	return conflicts
}

// Repair 检查并修复配置中缺失的值
func (c *AppConfig) Repair() {
	defaultCfg := GetDefaultConfig(c.WindowsAutoStart)

	if len(c.FanCurve) == 0 {
		c.FanCurve = defaultCfg.FanCurve
	} else {
		// 迁移：若曲线最高温度点 < 100°C，自动追加 100°C 点
		last := c.FanCurve[len(c.FanCurve)-1]
		if last.Temperature < 100 {
			c.FanCurve = append(c.FanCurve, FanCurvePoint{
				Temperature: 100,
				RPM:         last.RPM,
				Offset:      0,
			})
		}
	}
	if c.SmartStartStop == "" {
		c.SmartStartStop = defaultCfg.SmartStartStop
	}
	if c.TempUpdateRate <= 0 {
		c.TempUpdateRate = defaultCfg.TempUpdateRate
	}
	if c.TempSampleCount <= 0 {
		c.TempSampleCount = defaultCfg.TempSampleCount
	}
	if c.ManualGear == "" {
		c.ManualGear = defaultCfg.ManualGear
	}
	if c.ManualLevel == "" {
		c.ManualLevel = defaultCfg.ManualLevel
	}
	if c.CustomSpeedRPM <= 0 {
		c.CustomSpeedRPM = defaultCfg.CustomSpeedRPM
	} else {
		c.CustomSpeedRPM = NormalizeCustomSpeedRPM(c.CustomSpeedRPM)
	}
	if c.WindowWidth < DefaultWindowWidth || c.WindowHeight < DefaultWindowHeight {
		c.WindowWidth = defaultCfg.WindowWidth
		c.WindowHeight = defaultCfg.WindowHeight
	}
	if c.ThemeMode != "dark" && c.ThemeMode != "light" {
		c.ThemeMode = defaultCfg.ThemeMode
	}

	if c.RGBConfig == nil {
		c.RGBConfig = defaultCfg.RGBConfig
	} else {
		if c.RGBConfig.Mode == "" {
			c.RGBConfig.Mode = defaultCfg.RGBConfig.Mode
		}
		if len(c.RGBConfig.Colors) == 0 {
			c.RGBConfig.Colors = defaultCfg.RGBConfig.Colors
		}
		if c.RGBConfig.Speed == "" {
			c.RGBConfig.Speed = defaultCfg.RGBConfig.Speed
		}
		// 不对 Brightness 为 0 强行覆盖，因为可能用户就想设0亮度
	}
	if c.Hotkeys == nil {
		c.Hotkeys = defaultCfg.Hotkeys
	} else {
		c.Hotkeys.Global = repairHotkeyBindings(c.Hotkeys.Global, defaultCfg.Hotkeys.Global)
		c.Hotkeys.InApp = repairHotkeyBindings(c.Hotkeys.InApp, defaultCfg.Hotkeys.InApp)
	}
	c.NormalizeMonitorAutoStart()
}

func isObsoleteHotkeyAction(action string) bool {
	switch action {
	case "toggle-process-switch", "save-context", "escape-local-interaction", "toggle-sidebar":
		return true
	}
	return strings.HasPrefix(action, "navigate-")
}

func repairHotkeyBindings(current []HotkeyBinding, defaults []HotkeyBinding) []HotkeyBinding {
	index := map[string]HotkeyBinding{}
	orderedExtras := make([]HotkeyBinding, 0)
	defaultActions := map[string]bool{}
	for _, item := range defaults {
		defaultActions[item.Action] = true
	}
	for _, item := range current {
		if isObsoleteHotkeyAction(item.Action) {
			continue
		}
		index[item.Action] = item
		if !defaultActions[item.Action] {
			orderedExtras = append(orderedExtras, item)
		}
	}
	result := make([]HotkeyBinding, 0, len(defaults))
	for _, item := range defaults {
		existing, ok := index[item.Action]
		if !ok {
			result = append(result, item)
			continue
		}
		if existing.Accelerator == "" {
			existing.Accelerator = item.Accelerator
		}
		existing.Accelerator = NormalizeAccelerator(existing.Accelerator)
		if existing.Scope == "" {
			existing.Scope = item.Scope
		}
		existing.Editable = item.Editable
		existing.Category = item.Category
		existing.Description = item.Description
		result = append(result, existing)
	}
	for _, extra := range orderedExtras {
		extra.Accelerator = NormalizeAccelerator(extra.Accelerator)
		if extra.Scope == "" {
			extra.Scope = "global"
		}
		result = append(result, extra)
	}
	return result
}
