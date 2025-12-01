# Changelog

All notable changes to DailyFlow will be documented in this file.

## [1.0.0] - 2024-12-01

### Added
- 初始版本发布
- 录制功能：支持鼠标和键盘操作录制
- 回放功能：支持自动回放录制的操作
- 变速回放：支持 0.5x - 1.0x 速度调整
- 定时任务：支持每日定时自动执行
- 开机自启：支持 Windows 开机自动启动
- 热键支持：F8 录制，F12 回放
- 托盘功能：最小化到系统托盘
- DPI Awareness：支持高分屏准确录制和回放
- 单实例运行：防止重复启动
- 配置持久化：自动保存和恢复配置

### Technical Details
- 基于 Go 1.21+ 开发
- 使用 lxn/walk GUI 框架
- Windows Hook API 实现录制
- SendInput API 实现回放
- 零依赖，单文件绿色软件
- 文件大小 < 5MB

### Known Issues
- 不支持跨屏幕分辨率回放
- 密码输入以明文存储在 task.json
- 锁屏或休眠会中断回放

