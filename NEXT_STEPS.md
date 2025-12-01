# 下一步操作

## 1. 推送到 GitHub

```bash
cd /Users/yingdao/Documents/Project/日报2/DailyFlow
git init
git add .
git commit -m "Initial commit: DailyFlow v1.0.0"
git branch -M main
git remote add origin https://github.com/你的用户名/dailyflow.git
git push -u origin main
```

## 2. 触发构建

- 推送后 GitHub Actions 自动构建
- 5-10 分钟后在 Actions 页面下载 `DailyFlow.exe`

## 3. Windows 测试

- 下载 EXE 到 Windows 机器
- 按 `TESTING.md` 测试
- 完成

## 创建 Release（可选）

```bash
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

自动发布到 Releases 页面。

