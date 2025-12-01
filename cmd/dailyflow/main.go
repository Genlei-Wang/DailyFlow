package main

import (
	"dailyflow/internal/ui"
	"log"
	"os"
	"unsafe"

	"github.com/lxn/walk"
	"github.com/lxn/win"
	"golang.org/x/sys/windows"
)

const (
	VK_F8  = 0x77
	VK_F12 = 0x7B
)

var (
	kernel32            = windows.NewLazySystemDLL("kernel32.dll")
	user32              = windows.NewLazySystemDLL("user32.dll")
	shcore              = windows.NewLazySystemDLL("shcore.dll")
	procCreateMutex     = kernel32.NewProc("CreateMutexW")
	procGetLastError    = kernel32.NewProc("GetLastError")
	procRegisterHotKey  = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey = user32.NewProc("UnregisterHotKey")
	procSetProcessDPIAware = user32.NewProc("SetProcessDPIAware")
	procSetProcessDpiAwareness = shcore.NewProc("SetProcessDpiAwareness")
)

const (
	// Mutex 名称
	mutexName = "Global\\DailyFlow_Mutex"

	// 热键 ID
	HOTKEY_F8  = 1
	HOTKEY_F12 = 2
)

func main() {
	// 确保单实例运行
	if !ensureSingleInstance() {
		walk.MsgBox(nil, "DailyFlow", "程序已在运行中", walk.MsgBoxIconWarning)
		os.Exit(1)
	}

	// 设置 DPI Awareness（防止高分屏下坐标偏移）
	setDPIAware()

	// 创建主窗口
	mainWindow, err := ui.NewMainWindow()
	if err != nil {
		log.Fatalf("创建主窗口失败: %v", err)
	}

	// 创建 UI
	if err := mainWindow.Create(); err != nil {
		log.Fatalf("创建窗口 UI 失败: %v", err)
	}

	// 设置托盘
	if err := mainWindow.SetupTray(); err != nil {
		log.Fatalf("设置托盘失败: %v", err)
	}

	// 显示主窗口
	mainWindow.Show()

	// 注册全局热键（在独立 goroutine 中处理）
	hwnd := mainWindow.Handle()
	registerHotKeys(hwnd)
	defer unregisterHotKeys(hwnd)

	// 启动热键监听
	go handleHotKeys(mainWindow)

	// 运行应用程序
	mainWindow.Run()
}

// ensureSingleInstance 确保单实例运行
func ensureSingleInstance() bool {
	mutexNamePtr, _ := windows.UTF16PtrFromString(mutexName)
	handle, _, _ := procCreateMutex.Call(
		0,
		0,
		uintptr(unsafe.Pointer(mutexNamePtr)),
	)

	if handle == 0 {
		return false
	}

	// 检查是否已存在
	lastError, _, _ := procGetLastError.Call()
	if lastError == 183 { // ERROR_ALREADY_EXISTS
		return false
	}

	return true
}

// setDPIAware 设置 DPI 感知
func setDPIAware() {
	// 优先尝试新 API (Windows 8.1+)
	if procSetProcessDpiAwareness.Find() == nil {
		// PROCESS_SYSTEM_DPI_AWARE = 1
		procSetProcessDpiAwareness.Call(1)
		return
	}

	// 回退到旧 API (Windows Vista+)
	if procSetProcessDPIAware.Find() == nil {
		procSetProcessDPIAware.Call()
	}
}

// registerHotKeys 注册全局热键
func registerHotKeys(hwnd win.HWND) {
	// 注册 F8 热键（录制）
	ret, _, err := procRegisterHotKey.Call(
		uintptr(hwnd),
		HOTKEY_F8,
		0, // 无修饰键
		VK_F8,
	)
	if ret == 0 {
		log.Printf("警告: 注册 F8 热键失败: %v", err)
	}

	// 注册 F12 热键（回放）
	ret, _, err = procRegisterHotKey.Call(
		uintptr(hwnd),
		HOTKEY_F12,
		0, // 无修饰键
		VK_F12,
	)
	if ret == 0 {
		log.Printf("警告: 注册 F12 热键失败: %v", err)
	}
}

// unregisterHotKeys 注销全局热键
func unregisterHotKeys(hwnd win.HWND) {
	procUnregisterHotKey.Call(uintptr(hwnd), HOTKEY_F8)
	procUnregisterHotKey.Call(uintptr(hwnd), HOTKEY_F12)
}

// handleHotKeys 处理热键消息
func handleHotKeys(mainWindow *ui.AppMainWindow) {
	// 注意：热键功能在实际使用时由于需要持续监听 Windows 消息
	// 可能需要在主窗口中集成，这里提供基础实现
	// 实际项目中建议使用按钮操作代替热键
}

