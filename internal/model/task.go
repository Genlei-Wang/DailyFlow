package model

// TaskMeta 存储任务的元信息
type TaskMeta struct {
	Version     string `json:"version"`      // 数据版本，默认 "1.0"
	CreatedAt   int64  `json:"created_at"`   // 创建时间戳（Unix timestamp）
	Resolution  string `json:"resolution"`   // 录制时的屏幕分辨率（如 "1920x1080"）
	TotalEvents int    `json:"total_events"` // 总事件数量
}

// Event 表示单个录制的事件（鼠标或键盘）
type Event struct {
	Type    string `json:"type"`     // 事件类型: "mouse_move", "mouse_click", "key_press"
	X       int    `json:"x"`        // 鼠标 X 坐标（屏幕绝对坐标）
	Y       int    `json:"y"`        // 鼠标 Y 坐标（屏幕绝对坐标）
	Button  string `json:"button"`   // 鼠标按键: "left", "right", "middle", "double", "none"
	KeyCode int    `json:"key_code"` // 虚拟键码（VK_* 常量）
	Delay   int    `json:"delay"`    // 距离上一动作的毫秒数（Delta Time）
}

// TaskData 表示完整的任务数据结构（对应 task.json）
type TaskData struct {
	Meta   TaskMeta `json:"meta"`
	Events []Event  `json:"events"`
}

// NewTaskData 创建一个新的空任务数据
func NewTaskData(resolution string) *TaskData {
	return &TaskData{
		Meta: TaskMeta{
			Version:     "1.0",
			CreatedAt:   0, // 将在录制开始时设置
			Resolution:  resolution,
			TotalEvents: 0,
		},
		Events: make([]Event, 0),
	}
}

// AddEvent 添加一个事件到任务数据
func (t *TaskData) AddEvent(event Event) {
	t.Events = append(t.Events, event)
	t.Meta.TotalEvents = len(t.Events)
}

