package core

import (
	"dailyflow/internal/model"
	"dailyflow/internal/storage"
	"fmt"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	WH_MOUSE_LL    = 14
	WH_KEYBOARD_LL = 13

	WM_MOUSEMOVE   = 0x0200
	WM_LBUTTONDOWN = 0x0201
	WM_LBUTTONUP   = 0x0202
	WM_RBUTTONDOWN = 0x0204
	WM_RBUTTONUP   = 0x0205
	WM_MBUTTONDOWN = 0x0207
	WM_MBUTTONUP   = 0x0208
	WM_KEYDOWN     = 0x0100
	WM_KEYUP       = 0x0101
	WM_SYSKEYDOWN  = 0x0104
	WM_SYSKEYUP    = 0x0105
)

var (
	user32                  = windows.NewLazySystemDLL("user32.dll")
	procSetWindowsHookEx    = user32.NewProc("SetWindowsHookExW")
	procUnhookWindowsHookEx = user32.NewProc("UnhookWindowsHookEx")
	procCallNextHookEx      = user32.NewProc("CallNextHookEx")
	procGetMessage          = user32.NewProc("GetMessageW")
	procGetSystemMetrics    = user32.NewProc("GetSystemMetrics")
)

// MSLLHOOKSTRUCT 鼠标钩子结构
type MSLLHOOKSTRUCT struct {
	Pt          POINT
	MouseData   uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

// KBDLLHOOKSTRUCT 键盘钩子结构
type KBDLLHOOKSTRUCT struct {
	VkCode      uint32
	ScanCode    uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

// POINT 屏幕坐标点
type POINT struct {
	X, Y int32
}

// MSG Windows 消息结构
type MSG struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}

// Recorder 录制引擎
type Recorder struct {
	taskData          *model.TaskData
	mouseHook         uintptr
	keyboardHook      uintptr
	isRecording       bool
	lastEventTime     time.Time
	lastMousePos      POINT
	lastMouseMoveTime time.Time
	mutex             sync.Mutex
	stopChan          chan bool
}

// NewRecorder 创建新的录制器
func NewRecorder() *Recorder {
	return &Recorder{
		stopChan: make(chan bool),
	}
}

// StartRecording 开始录制
func (r *Recorder) StartRecording() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.isRecording {
		return fmt.Errorf("recording is already in progress")
	}

	// 获取屏幕分辨率
	width, _, _ := procGetSystemMetrics.Call(0)  // SM_CXSCREEN
	height, _, _ := procGetSystemMetrics.Call(1) // SM_CYSCREEN
	resolution := fmt.Sprintf("%dx%d", width, height)

	// 初始化任务数据
	r.taskData = model.NewTaskData(resolution)
	r.taskData.Meta.CreatedAt = time.Now().Unix()
	r.lastEventTime = time.Now()
	r.lastMouseMoveTime = time.Now()

	// 安装鼠标钩子
	mouseHook, err := r.setMouseHook()
	if err != nil {
		return fmt.Errorf("failed to install mouse hook: %w", err)
	}
	r.mouseHook = mouseHook

	// 安装键盘钩子
	keyboardHook, err := r.setKeyboardHook()
	if err != nil {
		r.unhookWindowsHookEx(r.mouseHook)
		return fmt.Errorf("failed to install keyboard hook: %w", err)
	}
	r.keyboardHook = keyboardHook

	r.isRecording = true

	// 启动消息循环（在独立 goroutine 中）
	go r.messageLoop()

	return nil
}

// StopRecording 停止录制并保存数据
func (r *Recorder) StopRecording() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if !r.isRecording {
		return fmt.Errorf("no recording in progress")
	}

	// 卸载钩子
	if r.mouseHook != 0 {
		r.unhookWindowsHookEx(r.mouseHook)
		r.mouseHook = 0
	}
	if r.keyboardHook != 0 {
		r.unhookWindowsHookEx(r.keyboardHook)
		r.keyboardHook = 0
	}

	r.isRecording = false

	// 保存任务数据
	if err := storage.SaveTask(r.taskData); err != nil {
		return fmt.Errorf("failed to save task: %w", err)
	}

	// 停止消息循环
	close(r.stopChan)

	return nil
}

// IsRecording 检查是否正在录制
func (r *Recorder) IsRecording() bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.isRecording
}

// setMouseHook 安装鼠标钩子
func (r *Recorder) setMouseHook() (uintptr, error) {
	hook, _, err := procSetWindowsHookEx.Call(
		uintptr(WH_MOUSE_LL),
		syscall.NewCallback(r.mouseProc),
		0,
		0,
	)
	if hook == 0 {
		return 0, fmt.Errorf("SetWindowsHookEx failed: %v", err)
	}
	return hook, nil
}

// setKeyboardHook 安装键盘钩子
func (r *Recorder) setKeyboardHook() (uintptr, error) {
	hook, _, err := procSetWindowsHookEx.Call(
		uintptr(WH_KEYBOARD_LL),
		syscall.NewCallback(r.keyboardProc),
		0,
		0,
	)
	if hook == 0 {
		return 0, fmt.Errorf("SetWindowsHookEx failed: %v", err)
	}
	return hook, nil
}

// unhookWindowsHookEx 卸载钩子
func (r *Recorder) unhookWindowsHookEx(hook uintptr) {
	if hook != 0 {
		procUnhookWindowsHookEx.Call(hook)
	}
}

// mouseProc 鼠标钩子回调
func (r *Recorder) mouseProc(nCode int, wParam uintptr, lParam uintptr) uintptr {
	if nCode >= 0 && r.isRecording {
		mouseInfo := (*MSLLHOOKSTRUCT)(unsafe.Pointer(lParam))
		now := time.Now()
		delay := int(now.Sub(r.lastEventTime).Milliseconds())

		switch wParam {
		case WM_MOUSEMOVE:
			// 限频采样：至少 50ms 间隔
			if now.Sub(r.lastMouseMoveTime) < 50*time.Millisecond {
				break
			}
			// 检查鼠标是否真的移动了（避免记录微小抖动）
			if mouseInfo.Pt.X == r.lastMousePos.X && mouseInfo.Pt.Y == r.lastMousePos.Y {
				break
			}
			r.lastMousePos = mouseInfo.Pt
			r.lastMouseMoveTime = now

			event := model.Event{
				Type:    "mouse_move",
				X:       int(mouseInfo.Pt.X),
				Y:       int(mouseInfo.Pt.Y),
				Button:  "none",
				KeyCode: 0,
				Delay:   delay,
			}
			r.taskData.AddEvent(event)
			r.lastEventTime = now

		case WM_LBUTTONDOWN:
			event := model.Event{
				Type:    "mouse_click",
				X:       int(mouseInfo.Pt.X),
				Y:       int(mouseInfo.Pt.Y),
				Button:  "left",
				KeyCode: 0,
				Delay:   delay,
			}
			r.taskData.AddEvent(event)
			r.lastEventTime = now

		case WM_RBUTTONDOWN:
			event := model.Event{
				Type:    "mouse_click",
				X:       int(mouseInfo.Pt.X),
				Y:       int(mouseInfo.Pt.Y),
				Button:  "right",
				KeyCode: 0,
				Delay:   delay,
			}
			r.taskData.AddEvent(event)
			r.lastEventTime = now

		case WM_MBUTTONDOWN:
			event := model.Event{
				Type:    "mouse_click",
				X:       int(mouseInfo.Pt.X),
				Y:       int(mouseInfo.Pt.Y),
				Button:  "middle",
				KeyCode: 0,
				Delay:   delay,
			}
			r.taskData.AddEvent(event)
			r.lastEventTime = now
		}
	}

	ret, _, _ := procCallNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
	return ret
}

// keyboardProc 键盘钩子回调
func (r *Recorder) keyboardProc(nCode int, wParam uintptr, lParam uintptr) uintptr {
	if nCode >= 0 && r.isRecording {
		if wParam == WM_KEYDOWN || wParam == WM_SYSKEYDOWN {
			kbInfo := (*KBDLLHOOKSTRUCT)(unsafe.Pointer(lParam))
			now := time.Now()
			delay := int(now.Sub(r.lastEventTime).Milliseconds())

			// 忽略 F8 键（录制控制键）
			if kbInfo.VkCode == 0x77 { // VK_F8
				ret, _, _ := procCallNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
				return ret
			}

			event := model.Event{
				Type:    "key_press",
				X:       0,
				Y:       0,
				Button:  "none",
				KeyCode: int(kbInfo.VkCode),
				Delay:   delay,
			}
			r.taskData.AddEvent(event)
			r.lastEventTime = now
		}
	}

	ret, _, _ := procCallNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
	return ret
}

// messageLoop Windows 消息循环
func (r *Recorder) messageLoop() {
	var msg MSG
	for {
		select {
		case <-r.stopChan:
			return
		default:
			// 非阻塞消息获取
			ret, _, _ := procGetMessage.Call(
				uintptr(unsafe.Pointer(&msg)),
				0,
				0,
				0,
			)
			if ret == 0 {
				return
			}
		}
	}
}

