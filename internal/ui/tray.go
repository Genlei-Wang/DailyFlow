package ui

import (
	"github.com/lxn/walk"
)

// SetupTray 设置系统托盘
func (mw *AppMainWindow) SetupTray() error {
	// 创建托盘图标
	ni, err := walk.NewNotifyIcon(mw)
	if err != nil {
		return err
	}

	mw.trayIcon = ni

	// 设置托盘图标（使用应用程序图标）
	icon, err := walk.NewIconFromResourceId(1) // 使用默认图标
	if err != nil {
		// 如果加载失败，使用系统默认图标
		icon, _ = walk.Resources.Icon("app.ico")
	}
	if icon != nil {
		ni.SetIcon(icon)
	}

	// 设置提示文本
	ni.SetToolTip("DailyFlow - 自动化助手")

	// 创建右键菜单
	if err := mw.createTrayMenu(); err != nil {
		return err
	}

	// 双击显示主窗口
	ni.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button == walk.LeftButton {
			mw.showMainWindow()
		}
	})

	// 显示托盘图标
	if err := ni.SetVisible(true); err != nil {
		return err
	}

	return nil
}

// createTrayMenu 创建托盘菜单
func (mw *AppMainWindow) createTrayMenu() error {
	// 显示主界面
	showAction := walk.NewAction()
	showAction.SetText("显示主界面")
	showAction.Triggered().Attach(func() {
		mw.showMainWindow()
	})

	// 关于
	aboutAction := walk.NewAction()
	aboutAction.SetText("关于")
	aboutAction.Triggered().Attach(func() {
		walk.MsgBox(mw, "关于 DailyFlow",
			"DailyFlow v1.0\n\n"+
				"Windows 自动化助手\n"+
				"轻量、简单、易用\n\n"+
				"支持录制和回放鼠标键盘操作\n"+
				"支持定时任务和开机自启动",
			walk.MsgBoxIconInformation)
	})

	// 退出
	exitAction := walk.NewAction()
	exitAction.SetText("退出程序")
	exitAction.Triggered().Attach(func() {
		if walk.MsgBox(mw, "确认退出", "确定要退出 DailyFlow 吗？", walk.MsgBoxYesNo|walk.MsgBoxIconQuestion) == walk.DlgCmdYes {
			mw.exitApplication()
		}
	})

	// 添加到托盘菜单
	if err := mw.trayIcon.ContextMenu().Actions().Add(showAction); err != nil {
		return err
	}
	if err := mw.trayIcon.ContextMenu().Actions().Add(aboutAction); err != nil {
		return err
	}
	if err := mw.trayIcon.ContextMenu().Actions().Add(walk.NewSeparatorAction()); err != nil {
		return err
	}
	if err := mw.trayIcon.ContextMenu().Actions().Add(exitAction); err != nil {
		return err
	}

	return nil
}

// showMainWindow 显示主窗口
func (mw *AppMainWindow) showMainWindow() {
	mw.Show()
	mw.BringToTop()
	mw.SetFocus()
}

// exitApplication 退出应用程序
func (mw *AppMainWindow) exitApplication() {
	// 停止调度器
	if mw.scheduler != nil && mw.scheduler.IsRunning() {
		mw.scheduler.Stop()
	}

	// 停止录制（如果正在进行）
	if mw.recorder != nil && mw.recorder.IsRecording() {
		mw.recorder.StopRecording()
	}

	// 停止回放（如果正在进行）
	if mw.player != nil && mw.player.IsPlaying() {
		mw.player.StopPlayback()
	}

	// 隐藏托盘图标
	if mw.trayIcon != nil {
		mw.trayIcon.Dispose()
	}

	// 退出程序
	walk.App().Exit(0)
}

