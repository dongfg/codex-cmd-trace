# Codex Command Trace

将 Codex 执行的命令和对应输出展示为桌面卡片。半透明科幻风格、逐字输入动画、平滑输出滚动，也可以把每条新命令展示在独立窗口中。

https://github.com/user-attachments/assets/96658b66-2ccd-40ee-8596-4289ce138591



## 下载

前往 **[Releases 下载 Windows 版本](https://github.com/dongfg/codex-cmd-trace/releases)**，选择 `codex-cmd-trace-<版本>-windows-amd64.zip`。

解压后双击 `CodexCommandTrace.exe` 即可启动，无需安装 Go 或编译器。目前提供 Windows x64 版本。ZIP 内附程序的 `SHA256SUMS.txt` 校验文件。

## 使用

先启动 Codex Command Trace，再正常使用 Codex。程序自动监听本地会话日志，将之后执行的命令与返回结果配对展示。

- **命令卡片**：新命令显示在顶部，失败命令显示为红色；长输出自动滚动。最多保留最近 20 条。
- **详情窗口**：点击卡片查看完整命令、输出及日志中提供的会话信息，并可复制内容。
- **置顶图钉**：切换主窗口置顶。
- **四格图标**：开启独立窗口模式，之后的新命令各用一个窗口，自动排列并支持多显示器；空间不足时淘汰最旧窗口。关闭后窗口依次淡出，保留的命令回到主窗口。
- **退出**：点击右上角 ×，同时关闭全部命令窗口。

默认监听 `%USERPROFILE%\.codex\sessions`；设置了 `CODEX_HOME` 时使用该目录下的 `sessions`。当前监听路径会显示在应用名称下方。

默认只展示启动后的新调用。输出出现的时机取决于 Codex 写入日志的时机，命令可能先显示为执行中，稍后再收到结果。

### 启动选项

在 PowerShell 中运行：

```powershell
# 指定会话日志目录
.\CodexCommandTrace.exe -dir 'D:\Codex\sessions'

# 同时读取启动前已有的历史记录
.\CodexCommandTrace.exe -history

# 播放模拟演示，不执行命令、不监听真实日志
.\CodexCommandTrace.exe -demo
```

演示依次播放 5 条固定命令、1 条等待 5 秒的命令，然后以随机 0.5～2 秒间隔播放 15 条模拟开发命令，最后输出“任务完成”。看到等待命令时，可以开启独立窗口模式进行录制。

## 注意事项

命令卡片仅预览长输出的尾部，完整已接收内容可在详情中查看。无法可靠拆分的复杂调用会按整组展示，不保证并行命令能逐条配对。日志可能包含本地路径、命令参数或敏感输出，录屏、截图和分享前请检查。

源码运行、测试、实现说明和发布流程见 [DEVELOPMENT.md](DEVELOPMENT.md)。
