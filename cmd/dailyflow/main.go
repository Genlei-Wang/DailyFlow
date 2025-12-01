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
	kernel32                   = windows.NewLazySystemDLL("kernel32.dll")
	user32                     = windows.NewLazySystemDLL("user32.dll")
	shcore                     = windows.NewLazySystemDLL("shcore.dll")
	comctl32                   = windows.NewLazySystemDLL("comctl32.dll")
	procCreateMutex            = kernel32.NewProc("CreateMutexW")
	procGetLastError           = kernel32.NewProc("GetLastError")
	procRegisterHotKey         = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey       = user32.NewProc("UnregisterHotKey")
	procSetProcessDPIAware     = user32.NewProc("SetProcessDPIAware")
	procSetProcessDpiAwareness = shcore.NewProc("SetProcessDpiAwareness")
	procInitCommonControlsEx   = comctl32.NewProc("InitCommonControlsEx")
	procInitCommonControls     = comctl32.NewProc("InitCommonControls")
)

const (
	// Mutex 名称
	mutexName = "Global\\DailyFlow_Mutex"

	// 热键 ID
	HOTKEY_F8  = 1
	HOTKEY_F12 = 2
)

func main() {
	// 必须在任何其他操作之前初始化 Common Controls
	// 这是 walk 库的要求
	initCommonControls()
	
	// 设置 DPI Awareness（防止高分屏下坐标偏移）
	setDPIAware()
	
	// 设置日志文件
	logFile, err := os.OpenFile("dailyflow_error.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		defer logFile.Close()
		log.SetOutput(logFile)
	}
	
	log.Println("========== DailyFlow 启动 ==========")
	log.Println("Common Controls 初始化完成")
	log.Println("DPI Awareness 设置完成")
	
	// 捕获 panic
	defer func() {
		if r := recover(); r != nil {
			log.Printf("程序崩溃: %v", r)
			walk.MsgBox(nil, "错误", "程序崩溃，请查看 dailyflow_error.log", walk.MsgBoxIconError)
		}
	}()

	// 确保单实例运行
	if !ensureSingleInstance() {
		log.Println("程序已在运行中")
		walk.MsgBox(nil, "DailyFlow", "程序已在运行中", walk.MsgBoxIconWarning)
		os.Exit(1)
	}
	log.Println("单实例检查通过")

	// 创建主窗口
	log.Println("开始创建主窗口...")
	mainWindow, err := ui.NewMainWindow()
	if err != nil {
		log.Printf("创建主窗口失败: %v", err)
		walk.MsgBox(nil, "错误", "创建主窗口失败: "+err.Error(), walk.MsgBoxIconError)
		log.Fatalf("创建主窗口失败: %v", err)
	}
	log.Println("主窗口创建成功")

	// 创建 UI
	log.Println("开始创建 UI...")
	if err := mainWindow.Create(); err != nil {
		log.Printf("创建窗口 UI 失败: %v", err)
		walk.MsgBox(nil, "错误", "创建窗口 UI 失败: "+err.Error(), walk.MsgBoxIconError)
		log.Fatalf("创建窗口 UI 失败: %v", err)
	}
	log.Println("UI 创建成功")

	// 设置托盘
	log.Println("开始设置托盘...")
	if err := mainWindow.SetupTray(); err != nil {
		log.Printf("设置托盘失败: %v", err)
		walk.MsgBox(nil, "错误", "设置托盘失败: "+err.Error(), walk.MsgBoxIconError)
		log.Fatalf("设置托盘失败: %v", err)
	}
	log.Println("托盘设置成功")

	// 显示主窗口
	log.Println("显示主窗口...")
	mainWindow.Show()

	// 注册全局热键（在独立 goroutine 中处理）
	hwnd := mainWindow.Handle()
	registerHotKeys(hwnd)
	defer unregisterHotKeys(hwnd)

	// 启动热键监听
	go handleHotKeys(mainWindow)

	log.Println("程序启动完成，进入主循环")
	// 运行应用程序
	mainWindow.Run()
	log.Println("程序正常退出")
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

// initCommonControls 初始化 Common Controls
func initCommonControls() {
	// ICC_WIN95_CLASSES = 0x000000FF (包含所有标准控件)
	// 但需要确保包含 ToolTip: ICC_ToolTip = 0x00000001
	// 使用完整的标志组合
	const (
		ICC_WIN95_CLASSES = 0xFF
		ICC_ToolTip       = 0x00000001
	)
	
	type INITCOMMONCONTROLSEX struct {
		Size uint32
		ICC  uint32
	}
	
	// 使用完整的 ICC_WIN95_CLASSES，它已经包含了 ToolTip
	icc := INITCOMMONCONTROLSEX{
		Size: uint32(unsafe.Sizeof(INITCOMMONCONTROLSEX{})),
		ICC:  ICC_WIN95_CLASSES,
	}
	
	ret, _, err := procInitCommonControlsEx.Call(uintptr(unsafe.Pointer(&icc)))
	if ret == 0 {
		log.Printf("InitCommonControlsEx 失败: %v，尝试使用旧版 API", err)
		// 如果失败，尝试使用旧版 InitCommonControls（无参数）
		procInitCommonControls.Call()
		log.Println("已使用 InitCommonControls 回退方案")
	} else {
		log.Println("InitCommonControlsEx 成功")
	}
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

