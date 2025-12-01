package model

// Config 表示应用配置结构（对应 config.json）
type Config struct {
	ScheduleTime string  `json:"schedule_time"` // 定时执行时间（格式："08:30"）
	IsEnabled    bool    `json:"is_enabled"`    // 是否启用定时任务
	AutoStart    bool    `json:"auto_start"`    // 是否开机自启动
	SpeedFactor  float64 `json:"speed_factor"`  // 播放速度因子（0.5=慢速, 1.0=原速）
	LastRunDate  string  `json:"last_run_date"` // 上次运行日期（格式："2023-12-01"）
}

// NewConfig 创建一个新的默认配置
func NewConfig() *Config {
	return &Config{
		ScheduleTime: "08:30",
		IsEnabled:    false,
		AutoStart:    false,
		SpeedFactor:  1.0,
		LastRunDate:  "",
	}
}

// HasTaskToday 检查今天是否已经运行过任务
func (c *Config) HasTaskToday(today string) bool {
	return c.LastRunDate == today
}

