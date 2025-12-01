package ui

import (
	"dailyflow/internal/core"
	"dailyflow/internal/model"
	"dailyflow/internal/storage"
	"fmt"
	"time"

	"github.com/lxn/walk"
	"github.com/lxn/walk/declarative"
)

// AppMainWindow 主窗口
type AppMainWindow struct {
	*walk.MainWindow
	recorder  *core.Recorder
	player    *core.Player
	scheduler *core.Scheduler
	trayIcon  *walk.NotifyIcon
	config    *model.Config

	// UI 控件
	statusLabel       *walk.Label
	recordBtn         *walk.PushButton
	playBtn           *walk.PushButton
	scheduleTimeEdit  *walk.LineEdit
	enableCheckBox    *walk.CheckBox
	speedSlider       *walk.Slider
	speedLabel        *walk.Label
	autoStartCheckBox *walk.CheckBox
}

// NewMainWindow 创建新的主窗口
func NewMainWindow() (*AppMainWindow, error) {
	mw := &AppMainWindow{
		recorder: core.NewRecorder(),
		player:   core.NewPlayer(),
	}
	mw.scheduler = core.NewScheduler(mw.player)

	// 加载配置
	config, err := storage.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	mw.config = config

	return mw, nil
}

// Create 创建并显示窗口
func (mw *AppMainWindow) Create() error {
	var statusLabel *walk.Label
	var recordBtn, playBtn *walk.PushButton
	var scheduleTimeEdit *walk.LineEdit
	var enableCheckBox, autoStartCheckBox *walk.CheckBox
	var speedSlider *walk.Slider
	var speedLabel *walk.Label

	// 使用声明式方式创建 UI
	err := (declarative.MainWindow{
		AssignTo: &mw.MainWindow,
		Title:    "DailyFlow",
		Size:     declarative.Size{Width: 320, Height: 480},
		Layout:   declarative.VBox{},
		Children: []declarative.Widget{
			// 警告横幅
			declarative.Composite{
				Background: declarative.SolidColorBrush{Color: walk.RGB(255, 255, 200)},
				Layout:     declarative.VBox{Margins: declarative.Margins{Left: 5, Top: 5, Right: 5, Bottom: 5}},
				Children: []declarative.Widget{
					declarative.Label{
						Text: "⚠️ 运行期间请保持屏幕常亮，勿锁屏",
						Font: declarative.Font{PointSize: 9},
					},
				},
			},

			// 状态显示
			declarative.Composite{
				Layout: declarative.VBox{Margins: declarative.Margins{Left: 10, Top: 15, Right: 10, Bottom: 10}},
				Children: []declarative.Widget{
					declarative.Label{
						AssignTo:  &statusLabel,
						Text:      "任务未配置",
						Font:      declarative.Font{PointSize: 12, Bold: true},
						Alignment: declarative.AlignHCenterVNear,
					},
				},
			},

			// 操作按钮
			declarative.Composite{
				Layout: declarative.HBox{Margins: declarative.Margins{Left: 10, Top: 10, Right: 10, Bottom: 10}},
				Children: []declarative.Widget{
					declarative.PushButton{
						AssignTo:  &recordBtn,
						Text:      "🔴 录制 (F8)",
						MinSize:   declarative.Size{Width: 130, Height: 40},
						OnClicked: func() { mw.onRecordClick() },
					},
					declarative.PushButton{
						AssignTo:  &playBtn,
						Text:      "🟢 回放 (F12)",
						MinSize:   declarative.Size{Width: 130, Height: 40},
						OnClicked: func() { mw.onPlayClick() },
					},
				},
			},

			// 配置区域
			declarative.GroupBox{
				Title:  "配置",
				Layout: declarative.VBox{Margins: declarative.Margins{Left: 10, Top: 10, Right: 10, Bottom: 10}},
				Children: []declarative.Widget{
					// 时间配置
					declarative.Composite{
						Layout: declarative.HBox{},
						Children: []declarative.Widget{
							declarative.Label{Text: "执行时间:", MinSize: declarative.Size{Width: 70}},
							declarative.LineEdit{
								AssignTo: &scheduleTimeEdit,
								Text:     mw.config.ScheduleTime,
								OnEditingFinished: func() {
									mw.onScheduleTimeChanged()
								},
							},
							declarative.CheckBox{
								AssignTo: &enableCheckBox,
								Text:     "每日启用",
								Checked:  mw.config.IsEnabled,
								OnClicked: func() {
									mw.onEnableChanged()
								},
							},
						},
					},

					// 速度控制
					declarative.Composite{
						Layout: declarative.HBox{Spacing: 5},
						Children: []declarative.Widget{
							declarative.Label{Text: "速度:", MinSize: declarative.Size{Width: 50}},
							declarative.Slider{
								AssignTo:       &speedSlider,
								MinValue:       50,
								MaxValue:       100,
								Value:          int(mw.config.SpeedFactor * 100),
								ToolTipText:    "调整回放速度",
								OnValueChanged: func() { mw.onSpeedChanged() },
							},
							declarative.Label{
								AssignTo: &speedLabel,
								Text:     fmt.Sprintf("%.1fx", mw.config.SpeedFactor),
								MinSize:  declarative.Size{Width: 40},
							},
						},
					},

					// 自启动
					declarative.CheckBox{
						AssignTo:  &autoStartCheckBox,
						Text:      "开机自启",
						Checked:   core.IsAutoStartEnabled(),
						OnClicked: func() { mw.onAutoStartChanged() },
					},
				},
			},
		},
	}).Create()

	if err != nil {
		return err
	}

	// 设置关闭事件（最小化到托盘）
	mw.MainWindow.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		*canceled = true
		mw.Hide()
	})

	// 保存控件引用
	mw.statusLabel = statusLabel
	mw.recordBtn = recordBtn
	mw.playBtn = playBtn
	mw.scheduleTimeEdit = scheduleTimeEdit
	mw.enableCheckBox = enableCheckBox
	mw.speedSlider = speedSlider
	mw.speedLabel = speedLabel
	mw.autoStartCheckBox = autoStartCheckBox

	// 更新状态显示
	mw.updateStatus()

	// 启动调度器
	if err := mw.scheduler.Start(); err != nil {
		walk.MsgBox(mw, "错误", fmt.Sprintf("启动调度器失败: %v", err), walk.MsgBoxIconError)
	}

	// 设置调度器回调
	mw.scheduler.SetCallbacks(
		func() {
			mw.Synchronize(func() {
				walk.MsgBox(mw, "任务执行", "定时任务已执行", walk.MsgBoxIconInformation)
				mw.updateStatus()
			})
		},
		func(err error) {
			mw.Synchronize(func() {
				walk.MsgBox(mw, "任务失败", fmt.Sprintf("任务执行失败: %v", err), walk.MsgBoxIconError)
			})
		},
	)

	return nil
}

// onRecordClick 录制按钮点击事件
func (mw *AppMainWindow) onRecordClick() {
	if mw.recorder.IsRecording() {
		// 停止录制
		if err := mw.recorder.StopRecording(); err != nil {
			walk.MsgBox(mw, "错误", fmt.Sprintf("停止录制失败: %v", err), walk.MsgBoxIconError)
			return
		}
		mw.recordBtn.SetText("🔴 录制 (F8)")
		walk.MsgBox(mw, "成功", "录制已保存", walk.MsgBoxIconInformation)
		mw.updateStatus()
	} else {
		// 开始录制
		if err := mw.recorder.StartRecording(); err != nil {
			walk.MsgBox(mw, "错误", fmt.Sprintf("开始录制失败: %v", err), walk.MsgBoxIconError)
			return
		}
		mw.recordBtn.SetText("⏹️ 停止录制 (F8)")
	}
}

// onPlayClick 回放按钮点击事件
func (mw *AppMainWindow) onPlayClick() {
	if mw.player.IsPlaying() {
		// 停止回放
		if err := mw.player.StopPlayback(); err != nil {
			walk.MsgBox(mw, "错误", fmt.Sprintf("停止回放失败: %v", err), walk.MsgBoxIconError)
			return
		}
		mw.playBtn.SetText("🟢 回放 (F12)")
	} else {
		// 开始回放
		speedFactor := float64(mw.speedSlider.Value()) / 100.0
		if err := mw.player.StartPlayback(speedFactor); err != nil {
			walk.MsgBox(mw, "错误", fmt.Sprintf("开始回放失败: %v", err), walk.MsgBoxIconError)
			return
		}
		mw.playBtn.SetText("⏹️ 停止回放 (F12)")
	}
}

// onScheduleTimeChanged 时间配置改变事件
func (mw *AppMainWindow) onScheduleTimeChanged() {
	newTime := mw.scheduleTimeEdit.Text()
	// 验证时间格式
	if _, err := time.Parse("15:04", newTime); err != nil {
		walk.MsgBox(mw, "错误", "时间格式错误，请使用 HH:MM 格式", walk.MsgBoxIconError)
		mw.scheduleTimeEdit.SetText(mw.config.ScheduleTime)
		return
	}

	mw.config.ScheduleTime = newTime
	mw.saveConfig()
	mw.updateStatus()
}

// onEnableChanged 启用状态改变事件
func (mw *AppMainWindow) onEnableChanged() {
	mw.config.IsEnabled = mw.enableCheckBox.Checked()
	mw.saveConfig()
	mw.updateStatus()
}

// onSpeedChanged 速度改变事件
func (mw *AppMainWindow) onSpeedChanged() {
	value := mw.speedSlider.Value()
	speedFactor := float64(value) / 100.0
	mw.config.SpeedFactor = speedFactor
	mw.speedLabel.SetText(fmt.Sprintf("%.1fx", speedFactor))
	mw.saveConfig()
}

// onAutoStartChanged 自启动改变事件
func (mw *AppMainWindow) onAutoStartChanged() {
	if mw.autoStartCheckBox.Checked() {
		if err := core.EnableAutoStart(); err != nil {
			walk.MsgBox(mw, "错误", fmt.Sprintf("启用自启动失败: %v", err), walk.MsgBoxIconError)
			mw.autoStartCheckBox.SetChecked(false)
			return
		}
		mw.config.AutoStart = true
	} else {
		if err := core.DisableAutoStart(); err != nil {
			walk.MsgBox(mw, "错误", fmt.Sprintf("禁用自启动失败: %v", err), walk.MsgBoxIconError)
			mw.autoStartCheckBox.SetChecked(true)
			return
		}
		mw.config.AutoStart = false
	}
	mw.saveConfig()
}

// saveConfig 保存配置
func (mw *AppMainWindow) saveConfig() {
	if err := storage.SaveConfig(mw.config); err != nil {
		walk.MsgBox(mw, "错误", fmt.Sprintf("保存配置失败: %v", err), walk.MsgBoxIconError)
	}
	if err := mw.scheduler.UpdateConfig(mw.config); err != nil {
		walk.MsgBox(mw, "错误", fmt.Sprintf("更新调度器配置失败: %v", err), walk.MsgBoxIconError)
	}
}

// updateStatus 更新状态显示
func (mw *AppMainWindow) updateStatus() {
	// 检查是否有任务数据
	taskData, err := storage.LoadTask()
	if err != nil || taskData == nil || len(taskData.Events) == 0 {
		mw.statusLabel.SetText("任务未配置")
		return
	}

	if mw.config.IsEnabled {
		now := time.Now()
		today := now.Format("2006-01-02")
		
		if mw.config.HasTaskToday(today) {
			mw.statusLabel.SetText("今日任务已完成")
		} else {
			// 计算下次运行时间
			scheduleTime, err := time.Parse("15:04", mw.config.ScheduleTime)
			if err == nil {
				targetTime := time.Date(
					now.Year(), now.Month(), now.Day(),
					scheduleTime.Hour(), scheduleTime.Minute(), 0, 0,
					now.Location(),
				)
				
				if now.After(targetTime) {
					// 今天的时间已过，显示明天
					targetTime = targetTime.Add(24 * time.Hour)
					mw.statusLabel.SetText(fmt.Sprintf("下次运行: 明天 %s", mw.config.ScheduleTime))
				} else {
					mw.statusLabel.SetText(fmt.Sprintf("下次运行: 今天 %s", mw.config.ScheduleTime))
				}
			}
		}
	} else {
		mw.statusLabel.SetText(fmt.Sprintf("任务已配置 (共 %d 个事件)", len(taskData.Events)))
	}
}

// Show 显示窗口
func (mw *AppMainWindow) Show() {
	mw.MainWindow.Show()
	mw.updateStatus()
}

// TriggerRecord 触发录制（用于热键）
func (mw *AppMainWindow) TriggerRecord() {
	mw.onRecordClick()
}

// TriggerPlay 触发回放（用于热键）
func (mw *AppMainWindow) TriggerPlay() {
	mw.onPlayClick()
}

