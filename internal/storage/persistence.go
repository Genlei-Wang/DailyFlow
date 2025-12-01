package storage

import (
	"dailyflow/internal/model"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	TaskFileName   = "task.json"
	ConfigFileName = "config.json"
)

// GetExecutableDir 获取可执行文件所在目录
func GetExecutableDir() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}
	return filepath.Dir(exePath), nil
}

// LoadTask 从 task.json 加载任务数据
func LoadTask() (*model.TaskData, error) {
	execDir, err := GetExecutableDir()
	if err != nil {
		return nil, err
	}

	taskPath := filepath.Join(execDir, TaskFileName)
	
	// 检查文件是否存在
	if _, err := os.Stat(taskPath); os.IsNotExist(err) {
		// 文件不存在，返回空任务数据
		return model.NewTaskData(""), nil
	}

	data, err := os.ReadFile(taskPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read task file: %w", err)
	}

	var taskData model.TaskData
	if err := json.Unmarshal(data, &taskData); err != nil {
		return nil, fmt.Errorf("failed to parse task file: %w", err)
	}

	return &taskData, nil
}

// SaveTask 保存任务数据到 task.json
func SaveTask(taskData *model.TaskData) error {
	execDir, err := GetExecutableDir()
	if err != nil {
		return err
	}

	taskPath := filepath.Join(execDir, TaskFileName)

	data, err := json.MarshalIndent(taskData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal task data: %w", err)
	}

	if err := os.WriteFile(taskPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write task file: %w", err)
	}

	return nil
}

// LoadConfig 从 config.json 加载配置
func LoadConfig() (*model.Config, error) {
	execDir, err := GetExecutableDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(execDir, ConfigFileName)

	// 检查文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// 文件不存在，返回默认配置
		return model.NewConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config model.Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// SaveConfig 保存配置到 config.json
func SaveConfig(config *model.Config) error {
	execDir, err := GetExecutableDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(execDir, ConfigFileName)

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config data: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

