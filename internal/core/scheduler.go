package core

import (
	"dailyflow/internal/model"
	"dailyflow/internal/storage"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	ole32            = windows.NewLazySystemDLL("ole32.dll")
	procCoInitialize = ole32.NewProc("CoInitialize")
)

// Scheduler 调度器
type Scheduler struct {
	config       *model.Config
	player       *Player
	isRunning    bool
	mutex        sync.Mutex
	stopChan     chan bool
	ticker       *time.Ticker
	onTaskRun    func() // UI 回调函数
	onTaskFailed func(error)
}

// NewScheduler 创建新的调度器
func NewScheduler(player *Player) *Scheduler {
	return &Scheduler{
		player:   player,
		stopChan: make(chan bool),
	}
}

// SetCallbacks 设置回调函数
func (s *Scheduler) SetCallbacks(onTaskRun func(), onTaskFailed func(error)) {
	s.onTaskRun = onTaskRun
	s.onTaskFailed = onTaskFailed
}

// Start 启动调度器
func (s *Scheduler) Start() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.isRunning {
		return fmt.Errorf("scheduler is already running")
	}

	// 加载配置
	config, err := storage.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	s.config = config

	s.isRunning = true

	// 启动心跳检测（每 60 秒检查一次）
	s.ticker = time.NewTicker(60 * time.Second)
	go s.heartbeatLoop()

	return nil
}

// Stop 停止调度器
func (s *Scheduler) Stop() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.isRunning {
		return fmt.Errorf("scheduler is not running")
	}

	s.isRunning = false
	if s.ticker != nil {
		s.ticker.Stop()
	}
	close(s.stopChan)

	return nil
}

// IsRunning 检查调度器是否运行
func (s *Scheduler) IsRunning() bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.isRunning
}

// UpdateConfig 更新配置
func (s *Scheduler) UpdateConfig(config *model.Config) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.config = config
	return storage.SaveConfig(config)
}

// heartbeatLoop 心跳循环
func (s *Scheduler) heartbeatLoop() {
	// 立即检查一次
	s.checkAndExecute()

	for {
		select {
		case <-s.stopChan:
			return
		case <-s.ticker.C:
			s.checkAndExecute()
		}
	}
}

// checkAndExecute 检查并执行任务
func (s *Scheduler) checkAndExecute() {
	s.mutex.Lock()
	config := s.config
	s.mutex.Unlock()

	if config == nil || !config.IsEnabled {
		return
	}

	now := time.Now()
	today := now.Format("2006-01-02")

	// 检查今天是否已经运行过
	if config.HasTaskToday(today) {
		return
	}

	// 解析设定的时间
	scheduleTime, err := time.Parse("15:04", config.ScheduleTime)
	if err != nil {
		if s.onTaskFailed != nil {
			s.onTaskFailed(fmt.Errorf("invalid schedule time: %w", err))
		}
		return
	}

	// 构造今天的执行时间
	targetTime := time.Date(
		now.Year(), now.Month(), now.Day(),
		scheduleTime.Hour(), scheduleTime.Minute(), 0, 0,
		now.Location(),
	)

	// 检查是否到达执行时间
	if now.Before(targetTime) {
		return
	}

	// 执行任务
	if err := s.executeTask(); err != nil {
		if s.onTaskFailed != nil {
			s.onTaskFailed(err)
		}
		return
	}

	// 更新最后运行日期
	config.LastRunDate = today
	if err := storage.SaveConfig(config); err != nil {
		if s.onTaskFailed != nil {
			s.onTaskFailed(fmt.Errorf("failed to save config: %w", err))
		}
	}

	// 回调通知 UI
	if s.onTaskRun != nil {
		s.onTaskRun()
	}
}

// executeTask 执行任务
func (s *Scheduler) executeTask() error {
	// 使用配置的速度因子
	speedFactor := s.config.SpeedFactor
	if speedFactor <= 0 {
		speedFactor = 1.0
	}

	return s.player.StartPlayback(speedFactor)
}

// EnableAutoStart 启用开机自启动
func EnableAutoStart() error {
	return createStartupShortcut(true)
}

// DisableAutoStart 禁用开机自启动
func DisableAutoStart() error {
	return createStartupShortcut(false)
}

// IsAutoStartEnabled 检查是否启用了开机自启动
func IsAutoStartEnabled() bool {
	shortcutPath := getStartupShortcutPath()
	_, err := os.Stat(shortcutPath)
	return err == nil
}

// getStartupShortcutPath 获取启动文件夹中的快捷方式路径
func getStartupShortcutPath() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
	}
	startupDir := filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs", "Startup")
	return filepath.Join(startupDir, "DailyFlow.lnk")
}

// createStartupShortcut 创建或删除启动快捷方式
func createStartupShortcut(enable bool) error {
	shortcutPath := getStartupShortcutPath()

	if !enable {
		// 删除快捷方式
		if err := os.Remove(shortcutPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove startup shortcut: %w", err)
		}
		return nil
	}

	// 获取当前可执行文件路径
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// 确保启动目录存在
	startupDir := filepath.Dir(shortcutPath)
	if err := os.MkdirAll(startupDir, 0755); err != nil {
		return fmt.Errorf("failed to create startup directory: %w", err)
	}

	// 创建快捷方式
	// 注意：这里使用 Windows Shell API 创建 .lnk 文件
	// 简化实现：直接调用 PowerShell 创建快捷方式
	return createShortcutViaPowerShell(shortcutPath, exePath)
}

// createShortcutViaPowerShell 通过 PowerShell 创建快捷方式
func createShortcutViaPowerShell(shortcutPath, targetPath string) error {
	// 使用 PowerShell 创建快捷方式（简化版本）
	// 构建 PowerShell 命令
	psCmd := fmt.Sprintf(
		`$WshShell = New-Object -ComObject WScript.Shell; $Shortcut = $WshShell.CreateShortcut('%s'); $Shortcut.TargetPath = '%s'; $Shortcut.Save()`,
		shortcutPath,
		targetPath,
	)

	// 执行 PowerShell 命令
	cmd := windows.StringToUTF16Ptr("powershell.exe")
	args := windows.StringToUTF16Ptr("-NoProfile -NonInteractive -Command " + psCmd)
	
	var si windows.StartupInfo
	var pi windows.ProcessInformation
	si.Cb = uint32(unsafe.Sizeof(si))

	err := windows.CreateProcess(
		cmd,
		args,
		nil,
		nil,
		false,
		windows.CREATE_NO_WINDOW,
		nil,
		nil,
		&si,
		&pi,
	)

	if err != nil {
		return fmt.Errorf("failed to create shortcut: %w", err)
	}

	// 等待进程完成
	windows.WaitForSingleObject(pi.Process, windows.INFINITE)
	windows.CloseHandle(pi.Process)
	windows.CloseHandle(pi.Thread)

	return nil
}

