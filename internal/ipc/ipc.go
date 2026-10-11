// Package ipc 提供核心服务与 GUI 之间的进程间通信
package ipc

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Microsoft/go-winio"
	"github.com/TIANLI0/BS2PRO-Controller/internal/types"
	"golang.org/x/sys/windows"
)

const (
	// PipeName 命名管道名称
	PipeName = "BS2PRO-Controller-IPC"
	// PipePath 命名管道完整路径
	PipePath = `\\.\pipe\` + PipeName

	// pipeSecurityDescriptor 只放行 SYSTEM、管理员，以及交互用户的读写。
	pipeSecurityDescriptor = "D:P(A;;GA;;;SY)(A;;GA;;;BA)(A;;GRGW;;;IU)"
	maxIPCFrameBytes       = 1 << 20
)

// RequestType 请求类型
type RequestType string

const (
	// 设备相关
	ReqConnect           RequestType = "Connect"
	ReqDisconnect        RequestType = "Disconnect"
	ReqGetDeviceStatus   RequestType = "GetDeviceStatus"
	ReqGetCurrentFanData RequestType = "GetCurrentFanData"

	// 配置相关
	ReqGetConfig             RequestType = "GetConfig"
	ReqUpdateConfig          RequestType = "UpdateConfig"
	ReqSetFanCurve           RequestType = "SetFanCurve"
	ReqGetFanCurve           RequestType = "GetFanCurve"
	ReqRegisterClient        RequestType = "RegisterClient"
	ReqNotifyImportCompleted RequestType = "NotifyImportCompleted"
	ReqNotifyExportCompleted RequestType = "NotifyExportCompleted"

	// 控制相关
	ReqSetAutoControl    RequestType = "SetAutoControl"
	ReqToggleAutoControl RequestType = "ToggleAutoControl"
	ReqSetManualGear     RequestType = "SetManualGear"
	ReqGetAvailableGears RequestType = "GetAvailableGears"
	ReqSetCustomSpeed    RequestType = "SetCustomSpeed"
	ReqSetGearLight      RequestType = "SetGearLight"
	ReqSetPowerOnStart   RequestType = "SetPowerOnStart"
	ReqSetSmartStartStop RequestType = "SetSmartStartStop"
	ReqSetBrightness     RequestType = "SetBrightness"

	// 温度相关
	ReqGetTemperature         RequestType = "GetTemperature"
	ReqTestTemperatureReading RequestType = "TestTemperatureReading"
	ReqTestBridgeProgram      RequestType = "TestBridgeProgram"
	ReqGetBridgeProgramStatus RequestType = "GetBridgeProgramStatus"

	// 调试相关
	ReqGetDebugInfo RequestType = "GetDebugInfo"
	ReqSetDebugMode RequestType = "SetDebugMode"

	// 系统相关
	ReqPing RequestType = "Ping"

	// 风扇偏移
	ReqApplyOffsetToCurve RequestType = "ApplyOffsetToCurve"
	ReqResetFanOffset     RequestType = "ResetFanOffset"
	ReqResetFanCurve      RequestType = "ResetFanCurve"

	// RGB 灯效控制
	ReqSetRGBMode            RequestType = "SetRGBMode"
	ReqCycleRGBMode          RequestType = "CycleRGBMode"
	ReqReportHotkeyConflicts RequestType = "ReportHotkeyConflicts"
	ReqUnsubscribeEvents     RequestType = "UnsubscribeEvents"
	ReqTriggerHotkeyAction   RequestType = "TriggerHotkeyAction"
	ReqSetHotkeyEditMode     RequestType = "SetHotkeyEditMode"

	// 服务管理
	ReqRestartService RequestType = "RestartService"
	ReqStopService    RequestType = "StopService"
)

// Request IPC 请求
type Request struct {
	Type RequestType     `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

// Response IPC 响应
type Response struct {
	IsResponse bool            `json:"isResponse"` // 标识这是响应而非事件
	Success    bool            `json:"success"`
	Error      string          `json:"error,omitempty"`
	Data       json.RawMessage `json:"data,omitempty"`
}

// Event IPC 事件（服务器推送给客户端）
type Event struct {
	IsEvent bool            `json:"isEvent"` // 标识这是事件
	Type    string          `json:"type"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// EventType 事件类型
const (
	EventFanDataUpdate       = "fan-data-update"
	EventTemperatureUpdate   = "temperature-update"
	EventDeviceConnected     = "device-connected"
	EventDeviceDisconnected  = "device-disconnected"
	EventDeviceError         = "device-error"
	EventConfigUpdate        = "config-update"
	EventFanCurveUpdate      = "fan-curve-update"
	EventCoreStatusUpdate    = "core-status-update"
	EventServiceConnected    = "service-connected"
	EventServiceDisconnected = "service-disconnected"
	EventNotificationRequest = "notification-request"
	EventHotkeyAction        = "hotkey-action"
	EventHotkeyEditMode      = "hotkey-edit-mode"
)

const (
	RoleGUI          = "gui"
	RoleMonitorAgent = "monitor"
)

type RegisterClientParams struct {
	Role string `json:"role"`
}

type NotifyProfilesParams struct {
	ProfileCount int `json:"profileCount"`
}

type TriggerHotkeyActionParams struct {
	Action string `json:"action"`
}

type ClientMeta struct {
	Role    string
	writeMu sync.Mutex
}

func writeFrame(meta *ClientMeta, conn net.Conn, payload []byte) error {
	if meta != nil {
		meta.writeMu.Lock()
		defer meta.writeMu.Unlock()
	}
	_ = conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	defer func() { _ = conn.SetWriteDeadline(time.Time{}) }()
	_, err := conn.Write(append(payload, '\n'))
	return err
}

func (s *Server) dropClient(conn net.Conn) bool {
	s.mutex.Lock()
	_, exists := s.clients[conn]
	if exists {
		delete(s.clients, conn)
	}
	s.mutex.Unlock()
	if exists {
		_ = conn.Close()
	}
	return exists
}

// Server IPC 服务器
type Server struct {
	listener net.Listener
	clients  map[net.Conn]*ClientMeta
	mutex    sync.RWMutex
	handler  RequestHandler
	logger   types.Logger
	running  atomic.Bool
}

// RequestHandler 请求处理函数类型
type RequestHandler func(req Request) Response

// NewServer 创建 IPC 服务器
func NewServer(handler RequestHandler, logger types.Logger) *Server {
	return &Server{
		clients: make(map[net.Conn]*ClientMeta),
		handler: handler,
		logger:  logger,
	}
}

// Start 启动服务器
func (s *Server) Start() error {
	// 创建命名管道监听器
	cfg := &winio.PipeConfig{
		SecurityDescriptor: pipeSecurityDescriptor,
	}

	listener, err := winio.ListenPipe(PipePath, cfg)
	if err != nil {
		return fmt.Errorf("创建命名管道失败: %v", err)
	}

	s.listener = listener
	s.running.Store(true)
	s.logInfo("IPC 服务器已启动", "pipe", PipePath)

	// 接受连接
	go s.acceptConnections()

	return nil
}

// acceptConnections 接受客户端连接
func (s *Server) acceptConnections() {
	defer func() {
		if r := recover(); r != nil {
			s.logError("acceptConnections panic", "error", r)
		}
	}()
	for s.running.Load() {
		conn, err := s.listener.Accept()
		if err != nil {
			if s.running.Load() {
				s.logWarn("接受 IPC 连接失败", "error", err)
				continue
			}
			return
		}

		role, err := authorizePipeClient(conn)
		if err != nil {
			s.logWarn("拒绝 IPC 连接", "error", err)
			_ = conn.Close()
			continue
		}

		s.mutex.Lock()
		s.clients[conn] = &ClientMeta{Role: role}
		s.mutex.Unlock()
		s.logInfo("IPC 客户端已允许", "role", role)

		go s.handleClient(conn)
	}
}

// handleClient 处理客户端连接
func (s *Server) handleClient(conn net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			stack := make([]byte, 4096)
			n := goruntime.Stack(stack, false)
			s.logError("handleClient panic", "error", r, "stack", string(stack[:n]))
		}
		s.mutex.Lock()
		delete(s.clients, conn)
		s.mutex.Unlock()
		_ = conn.Close()
	}()

	reader := bufio.NewReader(conn)

	for s.running.Load() {
		// 设置读取deadline若客户端 30 秒内无任何数据（包括心跳），
		// 视为僵尸连接，主动断开以释放 goroutine 和连接槽位。
		_ = conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		line, err := readIPCFrame(reader)
		if err != nil {
			if s.running.Load() {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					s.logDebug("IPC 读取超时", "reason", "client_unresponsive", "timeout", "30s", "error", netErr)
				} else if strings.Contains(err.Error(), "i/o timeout") {
					s.logDebug("IPC 读取超时", "reason", "io_timeout", "timeout", "30s", "error", err)
				} else if !strings.Contains(err.Error(), "EOF") {
					s.logDebug("IPC 读取失败", "error", err)
				}
			}
			return
		}
		// 读到数据后清除deadline，避免影响后续正常处理耗时
		_ = conn.SetReadDeadline(time.Time{})

		// 解析请求
		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			s.logError("解析 IPC 请求失败", "error", err)
			continue
		}
		resp := s.handler(req)
		resp.IsResponse = true

		// 发送响应
		respBytes, err := json.Marshal(resp)
		if err != nil {
			s.logError("序列化 IPC 响应失败", "error", err)
			continue
		}

		s.mutex.RLock()
		meta := s.clients[conn]
		s.mutex.RUnlock()
		if err = writeFrame(meta, conn, respBytes); err != nil {
			s.logError("发送 IPC 响应失败", "error", err)
			return
		}
	}
}

// BroadcastEvent 广播事件给所有客户端
func (s *Server) BroadcastEvent(eventType string, data any) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		s.logError("序列化 IPC 事件数据失败", "event", eventType, "error", err)
		return
	}

	event := Event{
		IsEvent: true, // 标记为事件
		Type:    eventType,
		Data:    dataBytes,
	}

	eventBytes, err := json.Marshal(event)
	if err != nil {
		s.logError("序列化 IPC 事件失败", "event", eventType, "error", err)
		return
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for conn, meta := range s.clients {
		role := ""
		if meta != nil {
			role = meta.Role
		}
		go func(c net.Conn, clientMeta *ClientMeta, clientRole string) {
			defer func() { _ = recover() }()
			if err := writeFrame(clientMeta, c, eventBytes); err != nil {
				if s.dropClient(c) {
					s.logWarn("发送 IPC 事件失败，已移除客户端", "role", clientRole, "event", eventType, "error", err)
				}
			}
		}(conn, meta, role)
	}
}

func (s *Server) BroadcastEventToRole(role, eventType string, data any) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		s.logError("序列化 IPC 事件数据失败", "role", role, "event", eventType, "error", err)
		return
	}

	event := Event{IsEvent: true, Type: eventType, Data: dataBytes}
	eventBytes, err := json.Marshal(event)
	if err != nil {
		s.logError("序列化 IPC 事件失败", "role", role, "event", eventType, "error", err)
		return
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for conn, meta := range s.clients {
		if meta == nil || meta.Role != role {
			continue
		}
		go func(c net.Conn, clientMeta *ClientMeta, clientRole string) {
			defer func() { _ = recover() }()
			if err := writeFrame(clientMeta, c, eventBytes); err != nil {
				if s.dropClient(c) {
					s.logWarn("发送定向 IPC 事件失败，已移除客户端", "role", clientRole, "event", eventType, "error", err)
				}
			}
		}(conn, meta, meta.Role)
	}
}

func (s *Server) HasRoleClient(role string) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	for _, meta := range s.clients {
		if meta != nil && meta.Role == role {
			return true
		}
	}
	return false
}

// Stop 停止服务器
func (s *Server) Stop() {
	s.running.Store(false)
	if s.listener != nil {
		_ = s.listener.Close()
	}

	s.mutex.Lock()
	for conn := range s.clients {
		_ = conn.Close()
	}
	s.clients = make(map[net.Conn]*ClientMeta)
	s.mutex.Unlock()

	s.logInfo("IPC 服务器已停止")
}

// HasClients 检查是否有客户端连接
func (s *Server) HasClients() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return len(s.clients) > 0
}

// 日志辅助方法
func (s *Server) logInfo(msg any, args ...any) {
	if s.logger != nil {
		s.logger.Info(msg, args...)
	}
}

func (s *Server) logError(msg any, args ...any) {
	if s.logger != nil {
		s.logger.Error(msg, args...)
	}
}

func (s *Server) logDebug(msg any, args ...any) {
	if s.logger != nil {
		s.logger.Debug(msg, args...)
	}
}

func (s *Server) logWarn(msg any, args ...any) {
	if s.logger != nil {
		s.logger.Warn(msg, args...)
	}
}

func readIPCFrame(r *bufio.Reader) ([]byte, error) {
	var buf bytes.Buffer
	for {
		chunk, err := r.ReadSlice('\n')
		if len(chunk) > 0 {
			if buf.Len()+len(chunk) > maxIPCFrameBytes {
				return nil, fmt.Errorf("IPC 帧超过 %d 字节", maxIPCFrameBytes)
			}
			buf.Write(chunk)
		}
		if err == nil {
			return buf.Bytes(), nil
		}
		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		return nil, err
	}
}

func authorizePipeClient(conn net.Conn) (string, error) {
	withFD, ok := conn.(interface{ Fd() uintptr })
	if !ok {
		return "", fmt.Errorf("IPC 连接不提供句柄")
	}
	var pid uint32
	if err := windows.GetNamedPipeClientProcessId(windows.Handle(withFD.Fd()), &pid); err != nil {
		return "", fmt.Errorf("读取 IPC 客户端进程失败: %w", err)
	}
	image, err := processImagePath(pid)
	if err != nil {
		return "", err
	}
	serverExe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("读取服务路径失败: %w", err)
	}
	return roleForClientImage(serverExe, image)
}

func processImagePath(pid uint32) (string, error) {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return "", fmt.Errorf("打开 IPC 客户端进程失败: %w", err)
	}
	defer func() { _ = windows.CloseHandle(handle) }()

	buf := make([]uint16, windows.MAX_PATH)
	for {
		n := uint32(len(buf))
		err = windows.QueryFullProcessImageName(handle, 0, &buf[0], &n)
		if err == nil {
			return windows.UTF16ToString(buf[:n]), nil
		}
		if err == windows.ERROR_INSUFFICIENT_BUFFER {
			next := int(n)
			if next <= len(buf) {
				next = len(buf) * 2
			}
			if next > 32768 {
				return "", fmt.Errorf("IPC 客户端路径过长")
			}
			buf = make([]uint16, next)
			continue
		}
		return "", fmt.Errorf("读取 IPC 客户端路径失败: %w", err)
	}
}

// implement: 身份只认服务 exe 同目录下的控制台和 Monitor。这两份进程被注入后仍能发请求。
func roleForClientImage(serverExe, clientImage string) (string, error) {
	var name, role string
	switch strings.ToLower(filepath.Base(clientImage)) {
	case "bs2pro-controller.exe":
		name, role = "BS2PRO-Controller.exe", RoleGUI
	case "bs2pro-monitor.exe":
		name, role = "BS2PRO-Monitor.exe", RoleMonitorAgent
	default:
		return "", fmt.Errorf("拒绝未知 IPC 客户端 %s", clientImage)
	}
	expected := filepath.Join(filepath.Dir(serverExe), name)
	if !sameFilePath(clientImage, expected) {
		return "", fmt.Errorf("拒绝安装目录之外的 IPC 客户端 %s", clientImage)
	}
	return role, nil
}

func sameFilePath(a, b string) bool {
	left, errLeft := filepath.Abs(longPath(a))
	right, errRight := filepath.Abs(longPath(b))
	if errLeft != nil || errRight != nil {
		return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
	}
	return strings.EqualFold(filepath.Clean(left), filepath.Clean(right))
}

func longPath(path string) string {
	ptr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return path
	}
	buf := make([]uint16, windows.MAX_PATH)
	for {
		n, err := windows.GetLongPathName(ptr, &buf[0], uint32(len(buf)))
		if err == nil && n > 0 && n < uint32(len(buf)) {
			return windows.UTF16ToString(buf[:n])
		}
		if n > uint32(len(buf)) && n < 32768 {
			buf = make([]uint16, n)
			continue
		}
		return path
	}
}

// Client IPC 客户端
type Client struct {
	conn           net.Conn
	mutex          sync.Mutex
	reader         *bufio.Reader
	logger         types.Logger
	eventHandler   func(Event)
	responseChan   chan *Response
	connected      bool
	connMutex      sync.RWMutex
	connGeneration int64
	role           string
}

type serviceEventPayload struct {
	Timestamp  string `json:"timestamp,omitempty"`
	Reason     string `json:"reason,omitempty"`
	Generation int64  `json:"generation,omitempty"`
}

// NewClient 创建 IPC 客户端
func NewClient(logger types.Logger) *Client {
	return &Client{
		logger:       logger,
		responseChan: make(chan *Response, 1),
	}
}

// Connect 连接到服务器
func (c *Client) Connect() error {
	c.connMutex.Lock()
	defer c.connMutex.Unlock()

	// 如果已有连接，先关闭旧连接
	if c.connected && c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
		c.reader = nil
		c.connected = false
	}

	timeout := 5 * time.Second
	conn, err := winio.DialPipe(PipePath, &timeout)
	if err != nil {
		return fmt.Errorf("连接 IPC 服务器失败: %v", err)
	}

	c.conn = conn
	c.reader = bufio.NewReader(conn)
	c.connected = true
	// 递增generation：旧readLoop检测到generation变化后会主动退出，
	// 确保任意时刻只有一个readLoop goroutine在运行。
	gen := atomic.AddInt64(&c.connGeneration, 1)
	c.logInfo("已连接到 IPC 服务器", "connected", true, "role", c.role)

	// 启动消息接收循环
	go c.readLoop(gen)

	// 触发服务连接事件
	if c.eventHandler != nil {
		payload, _ := json.Marshal(serviceEventPayload{
			Timestamp:  time.Now().Format(time.RFC3339),
			Generation: gen,
		})
		event := Event{
			IsEvent: true,
			Type:    EventServiceConnected,
			Data:    payload,
		}
		go c.eventHandler(event)
	}

	return nil
}

// readLoop 统一读取消息循环
// gen 表示连接代数，变化时主动退出旧循环
func (c *Client) readLoop(gen int64) {
	c.logInfo("IPC 读取循环已启动", "generation", gen)
	for {
		// 连接代数变化时退出旧循环
		if atomic.LoadInt64(&c.connGeneration) != gen {
			c.logInfo("IPC 读取循环已退出", "generation", gen, "reason", "new_connection")
			return
		}

		c.connMutex.RLock()
		if !c.connected || c.reader == nil {
			c.connMutex.RUnlock()
			c.logInfo("IPC 读取循环已退出", "generation", gen, "connected", false, "reason", "connection_unavailable")
			return
		}
		reader := c.reader
		c.connMutex.RUnlock()

		line, err := readIPCFrame(reader)
		if err != nil {
			// 若已被新连接取代则直接退出
			if atomic.LoadInt64(&c.connGeneration) != gen {
				c.logInfo("IPC 读取循环已退出", "generation", gen, "reason", "superseded_after_read_error")
				return
			}
			c.logInfo("读取 IPC 消息失败，连接可能已断开", "generation", gen, "error", err)
			c.connMutex.Lock()
			c.connected = false
			if c.conn != nil {
				_ = c.conn.Close()
				c.conn = nil
			}
			c.reader = nil
			c.connMutex.Unlock()
			c.logInfo("已标记 IPC 连接断开", "generation", gen, "connected", false)

			// 触发服务断开事件
			if c.eventHandler != nil {
				payload, _ := json.Marshal(serviceEventPayload{
					Reason:     err.Error(),
					Generation: gen,
				})
				event := Event{
					IsEvent: true,
					Type:    EventServiceDisconnected,
					Data:    payload,
				}
				go c.eventHandler(event)
			}
			return
		}

		// 使用通用结构来检测消息类型
		var msg struct {
			IsResponse bool `json:"isResponse"`
			IsEvent    bool `json:"isEvent"`
		}
		if err := json.Unmarshal(line, &msg); err != nil {
			c.logDebug("解析 IPC 消息类型失败", "error", err)
			continue
		}

		if msg.IsResponse {
			var resp Response
			if err := json.Unmarshal(line, &resp); err == nil {
				select {
				case c.responseChan <- &resp:
				default:
					c.logDebug("IPC 响应通道已满，丢弃响应")
				}
			}
		} else if msg.IsEvent {
			var event Event
			if err := json.Unmarshal(line, &event); err == nil && event.Type != "" {
				if c.eventHandler != nil {
					go c.eventHandler(event)
				}
			}
		}
	}
}

// SetEventHandler 设置事件处理函数
func (c *Client) SetEventHandler(handler func(Event)) {
	c.eventHandler = handler
}

func (c *Client) SetRole(role string) {
	c.role = role
}

// SendRequest 发送请求并等待响应
func (c *Client) SendRequest(reqType RequestType, data any) (*Response, error) {
	c.connMutex.RLock()
	needsConnect := !c.connected || c.conn == nil
	c.connMutex.RUnlock()

	c.logInfo("准备发送 IPC 请求", "type", reqType, "connected", !needsConnect)

	if needsConnect {
		// 先连接后加请求锁，避免阻塞其他请求方
		c.logInfo("尝试连接 IPC 服务器", "type", reqType)
		if err := c.Connect(); err != nil {
			c.logInfo("连接 IPC 服务器失败", "type", reqType, "error", err)
			return nil, fmt.Errorf("未连接到服务器: %v", err)
		}
		c.logInfo("IPC 服务器连接成功", "type", reqType)
	}

	var dataBytes json.RawMessage
	if data != nil {
		var err error
		dataBytes, err = json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("序列化请求数据失败: %v", err)
		}
	}
	req := Request{Type: reqType, Data: dataBytes}
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %v", err)
	}

	// 保证请求与响应一一配对
	c.mutex.Lock()
	defer c.mutex.Unlock()

	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		// 获取当前连接快照
		c.connMutex.RLock()
		connected := c.connected
		conn := c.conn
		c.connMutex.RUnlock()

		c.logInfo("发送 IPC 请求尝试", "type", reqType, "attempt", attempt+1, "connected", connected, "conn_present", conn != nil)

		if !connected || conn == nil {
			// 断连后尝试重连
			c.logWarn("IPC 连接已断开，尝试重连", "type", reqType, "attempt", attempt+1, "connected", connected)
			if err := c.Connect(); err != nil {
				lastErr = fmt.Errorf("重连失败: %v", err)
				c.logWarn("IPC 重连失败", "type", reqType, "attempt", attempt+1, "error", err)
				continue
			}
			c.connMutex.RLock()
			conn = c.conn
			c.connMutex.RUnlock()
			if conn == nil {
				lastErr = fmt.Errorf("重连后连接仍为空")
				c.logError("IPC 重连后连接为空", "type", reqType, "attempt", attempt+1)
				continue
			}
			c.logInfo("IPC 重连成功", "type", reqType, "attempt", attempt+1)
		}

		// 清空旧响应
		select {
		case <-c.responseChan:
		default:
		}

		_, err = conn.Write(append(reqBytes, '\n'))
		if err != nil {
			lastErr = err
			c.logWarn("发送 IPC 请求失败", "type", reqType, "attempt", attempt+1, "error", err)
			// 标记断连，下次循环重连
			c.connMutex.Lock()
			c.connected = false
			if c.conn != nil {
				_ = c.conn.Close()
				c.conn = nil
			}
			c.reader = nil
			c.connMutex.Unlock()
			if attempt == 1 {
				break
			}
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// 等待响应
		select {
		case resp := <-c.responseChan:
			return resp, nil
		case <-time.After(10 * time.Second):
			return nil, fmt.Errorf("等待响应超时")
		}
	}

	return nil, fmt.Errorf("发送请求失败: %v", lastErr)
}

// Close 关闭连接
func (c *Client) Close() {
	c.connMutex.Lock()
	defer c.connMutex.Unlock()

	c.connected = false
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
	c.reader = nil
}

// IsConnected 检查是否已连接
func (c *Client) IsConnected() bool {
	c.connMutex.RLock()
	defer c.connMutex.RUnlock()
	return c.connected && c.conn != nil && c.reader != nil
}

func (c *Client) ConnectionGeneration() int64 {
	if !c.IsConnected() {
		return 0
	}
	return atomic.LoadInt64(&c.connGeneration)
}

func (c *Client) LatestConnectionGeneration() int64 {
	return atomic.LoadInt64(&c.connGeneration)
}

// 日志辅助方法
func (c *Client) logInfo(msg any, args ...any) {
	if c.logger != nil {
		c.logger.Info(msg, args...)
	}
}

func (c *Client) logDebug(msg any, args ...any) {
	if c.logger != nil {
		c.logger.Debug(msg, args...)
	}
}

func (c *Client) logError(msg any, args ...any) {
	if c.logger != nil {
		c.logger.Error(msg, args...)
	}
}

func (c *Client) logWarn(msg any, args ...any) {
	if c.logger != nil {
		c.logger.Warn(msg, args...)
	}
}

// SetAutoControlParams 设置智能变频参数
type SetAutoControlParams struct {
	Enabled bool `json:"enabled"`
}

// SetManualGearParams 设置手动挡位参数
type SetManualGearParams struct {
	Gear  string `json:"gear"`
	Level string `json:"level"`
}

// SetCustomSpeedParams 设置自定义转速参数
type SetCustomSpeedParams struct {
	Enabled bool `json:"enabled"`
	RPM     int  `json:"rpm"`
}

// SetBoolParams 布尔参数
type SetBoolParams struct {
	Enabled bool `json:"enabled"`
}

// SetStringParams 字符串参数
type SetStringParams struct {
	Value string `json:"value"`
}

// SetIntParams 整数参数
type SetIntParams struct {
	Value int `json:"value"`
}

// RGBColorParam RGB颜色参数
type RGBColorParam struct {
	R int `json:"r"`
	G int `json:"g"`
	B int `json:"b"`
}

// SetRGBModeParams RGB灯效模式参数
type SetRGBModeParams struct {
	Mode       string          `json:"mode"`       // smart/rotation/breathing/static_single/static_multi/flowing/off
	Colors     []RGBColorParam `json:"colors"`     // 颜色列表
	Speed      string          `json:"speed"`      // fast/medium/slow
	Brightness int             `json:"brightness"` // 0-100
}
