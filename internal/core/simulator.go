package core

import (
	"dailyflow/internal/model"
	"dailyflow/internal/storage"
	"fmt"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	INPUT_MOUSE    = 0
	INPUT_KEYBOARD = 1

	MOUSEEVENTF_MOVE       = 0x0001
	MOUSEEVENTF_LEFTDOWN   = 0x0002
	MOUSEEVENTF_LEFTUP     = 0x0004
	MOUSEEVENTF_RIGHTDOWN  = 0x0008
	MOUSEEVENTF_RIGHTUP    = 0x0010
	MOUSEEVENTF_MIDDLEDOWN = 0x0020
	MOUSEEVENTF_MIDDLEUP   = 0x0040
	MOUSEEVENTF_ABSOLUTE   = 0x8000

	KEYEVENTF_KEYUP = 0x0002
)

var (
	user32              = windows.NewLazySystemDLL("user32.dll")
	procSendInput        = user32.NewProc("SendInput")
	procSetCursorPos     = user32.NewProc("SetCursorPos")
	procGetCursorPos     = user32.NewProc("GetCursorPos")
	procGetAsyncKeyState = user32.NewProc("GetAsyncKeyState")
)

// INPUT Windows 输入结构
type INPUT struct {
	Type uint32
	Mi   MOUSEINPUT
	Ki   KEYBDINPUT
	Hi   HARDWAREINPUT
}

// MOUSEINPUT 鼠标输入结构
type MOUSEINPUT struct {
	Dx          int32
	Dy          int32
	MouseData   uint32
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

// KEYBDINPUT 键盘输入结构
type KEYBDINPUT struct {
	WVk         uint16
	WScan       uint16
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
	Padding     [8]byte
}

// HARDWAREINPUT 硬件输入结构
type HARDWAREINPUT struct {
	UMsg    uint32
	WParamL uint16
	WParamH uint16
}

// Player 回放引擎
type Player struct {
	taskData      *model.TaskData
	isPlaying     bool
	isPaused      bool
	speedFactor   float64
	mutex         sync.Mutex
	stopChan      chan bool
	pauseChan     chan bool
	initialCursor POINT
}

// NewPlayer 创建新的回放器
func NewPlayer() *Player {
	return &Player{
		speedFactor: 1.0,
		stopChan:    make(chan bool, 1),
		pauseChan:   make(chan bool, 1),
	}
}

// StartPlayback 开始回放
func (p *Player) StartPlayback(speedFactor float64) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.isPlaying {
		return fmt.Errorf("playback is already in progress")
	}

	// 加载任务数据
	taskData, err := storage.LoadTask()
	if err != nil {
		return fmt.Errorf("failed to load task: %w", err)
	}

	if taskData == nil || len(taskData.Events) == 0 {
		return fmt.Errorf("no task data to play")
	}

	p.taskData = taskData
	p.speedFactor = speedFactor
	p.isPlaying = true
	p.isPaused = false

	// 记录初始鼠标位置
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&p.initialCursor)))

	// 在独立 goroutine 中执行回放
	go p.playbackLoop()

	return nil
}

// StopPlayback 停止回放
func (p *Player) StopPlayback() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.isPlaying {
		return fmt.Errorf("no playback in progress")
	}

	p.isPlaying = false
	p.stopChan <- true

	return nil
}

// PausePlayback 暂停回放
func (p *Player) PausePlayback() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.isPlaying {
		return fmt.Errorf("no playback in progress")
	}

	if p.isPaused {
		return fmt.Errorf("playback is already paused")
	}

	p.isPaused = true
	return nil
}

// ResumePlayback 恢复回放
func (p *Player) ResumePlayback() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.isPlaying {
		return fmt.Errorf("no playback in progress")
	}

	if !p.isPaused {
		return fmt.Errorf("playback is not paused")
	}

	p.isPaused = false
	p.pauseChan <- true
	return nil
}

// IsPlaying 检查是否正在回放
func (p *Player) IsPlaying() bool {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.isPlaying
}

// playbackLoop 回放循环
func (p *Player) playbackLoop() {
	defer func() {
		p.mutex.Lock()
		p.isPlaying = false
		p.mutex.Unlock()
	}()

	for i, event := range p.taskData.Events {
		// 检查是否需要停止
		select {
		case <-p.stopChan:
			return
		default:
		}

		// 检查是否暂停
		for p.isPaused {
			select {
			case <-p.pauseChan:
				// 继续执行
			case <-p.stopChan:
				return
			case <-time.After(100 * time.Millisecond):
				// 继续检查暂停状态
			}
		}

		// 检测用户物理鼠标移动
		var currentCursor POINT
		procGetCursorPos.Call(uintptr(unsafe.Pointer(&currentCursor)))
		
		// 如果鼠标移动超过 50px，暂停并警告
		dx := currentCursor.X - p.initialCursor.X
		dy := currentCursor.Y - p.initialCursor.Y
		distance := dx*dx + dy*dy
		if i > 0 && distance > 50*50 { // 50px 阈值
			p.mutex.Lock()
			p.isPaused = true
			p.mutex.Unlock()
			// 这里应该触发 UI 警告，但现在只是暂停
			// UI 层需要轮询检查 isPaused 状态
			continue
		}

		// 计算延迟（考虑速度因子）
		if event.Delay > 0 {
			actualDelay := time.Duration(float64(event.Delay)/p.speedFactor) * time.Millisecond
			time.Sleep(actualDelay)
		}

		// 执行事件
		if err := p.executeEvent(&event); err != nil {
			// 记录错误，但继续执行
			fmt.Printf("Error executing event: %v\n", err)
		}

		// 更新初始位置
		p.initialCursor = currentCursor
	}
}

// executeEvent 执行单个事件
func (p *Player) executeEvent(event *model.Event) error {
	switch event.Type {
	case "mouse_move":
		return p.simulateMouseMove(event.X, event.Y)
	case "mouse_click":
		return p.simulateMouseClick(event.X, event.Y, event.Button)
	case "key_press":
		return p.simulateKeyPress(event.KeyCode)
	default:
		return fmt.Errorf("unknown event type: %s", event.Type)
	}
}

// simulateMouseMove 模拟鼠标移动
func (p *Player) simulateMouseMove(x, y int) error {
	ret, _, err := procSetCursorPos.Call(uintptr(x), uintptr(y))
	if ret == 0 {
		return fmt.Errorf("SetCursorPos failed: %v", err)
	}
	return nil
}

// simulateMouseClick 模拟鼠标点击
func (p *Player) simulateMouseClick(x, y int, button string) error {
	// 先移动鼠标到目标位置
	if err := p.simulateMouseMove(x, y); err != nil {
		return err
	}

	// 小延迟，确保移动完成
	time.Sleep(10 * time.Millisecond)

	var downFlag, upFlag uint32
	switch button {
	case "left":
		downFlag = MOUSEEVENTF_LEFTDOWN
		upFlag = MOUSEEVENTF_LEFTUP
	case "right":
		downFlag = MOUSEEVENTF_RIGHTDOWN
		upFlag = MOUSEEVENTF_RIGHTUP
	case "middle":
		downFlag = MOUSEEVENTF_MIDDLEDOWN
		upFlag = MOUSEEVENTF_MIDDLEUP
	case "double":
		// 双击：两次左键点击
		if err := p.simulateMouseClick(x, y, "left"); err != nil {
			return err
		}
		time.Sleep(50 * time.Millisecond)
		return p.simulateMouseClick(x, y, "left")
	default:
		return fmt.Errorf("unknown button: %s", button)
	}

	// 按下
	input := INPUT{
		Type: INPUT_MOUSE,
		Mi: MOUSEINPUT{
			DwFlags: downFlag,
		},
	}
	ret, _, err := procSendInput.Call(1, uintptr(unsafe.Pointer(&input)), unsafe.Sizeof(input))
	if ret == 0 {
		return fmt.Errorf("SendInput (mouse down) failed: %v", err)
	}

	// 小延迟
	time.Sleep(10 * time.Millisecond)

	// 释放
	input.Mi.DwFlags = upFlag
	ret, _, err = procSendInput.Call(1, uintptr(unsafe.Pointer(&input)), unsafe.Sizeof(input))
	if ret == 0 {
		return fmt.Errorf("SendInput (mouse up) failed: %v", err)
	}

	return nil
}

// simulateKeyPress 模拟按键
func (p *Player) simulateKeyPress(keyCode int) error {
	// 按下
	input := INPUT{
		Type: INPUT_KEYBOARD,
		Ki: KEYBDINPUT{
			WVk: uint16(keyCode),
		},
	}
	ret, _, err := procSendInput.Call(1, uintptr(unsafe.Pointer(&input)), unsafe.Sizeof(input))
	if ret == 0 {
		return fmt.Errorf("SendInput (key down) failed: %v", err)
	}

	// 小延迟
	time.Sleep(10 * time.Millisecond)

	// 释放
	input.Ki.DwFlags = KEYEVENTF_KEYUP
	ret, _, err = procSendInput.Call(1, uintptr(unsafe.Pointer(&input)), unsafe.Sizeof(input))
	if ret == 0 {
		return fmt.Errorf("SendInput (key up) failed: %v", err)
	}

	return nil
}

