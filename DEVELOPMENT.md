# 开发说明

Codex Command Trace 使用 Go + Fyne，监听 Codex 本地会话 JSONL，将命令与对应输出发送给桌面界面或控制台。

## 环境与启动

Go 版本以 `go.mod` 为准。Fyne 桌面构建需要开启 CGO，并安装 C 编译器及平台图形开发依赖，参见 [Fyne 环境准备](https://docs.fyne.io/started/quick/)。目前仅在 Windows 实际验证；原生圆角、透明度、置顶、拖动和独立窗口定位后端已适配 Windows，其他平台的独立窗口模式暂不支持。

Windows 开发启动：

```powershell
.\run.ps1
.\run.ps1 -Demo
.\run.ps1 -Cli
.\run.ps1 -History
.\run.ps1 -Dir 'D:\Codex\sessions'
```

脚本启用 CGO；若 PATH 中没有 GCC，会尝试使用 `%LOCALAPPDATA%\codex-cmd-trace\tools\w64devkit\bin\gcc.exe`。该便携编译器需要自行准备，脚本不会自动安装。

也可手动运行：

```powershell
$env:CGO_ENABLED = '1'
# 如 GCC 不在 PATH 中，先设置 $env:CC 为 gcc.exe 的完整路径。
go run .
go run . -cli
go run . -history
go run . -dir 'D:\Codex\sessions' -interval 3s
```

控制台模式可以通过 Ctrl+C 停止；发布版采用 Windows GUI 子系统，控制台调试建议使用 `go run . -cli`。

## 验证与构建

```powershell
go mod verify
go test -tags ci ./...
go vet -tags ci ./...
New-Item -ItemType Directory -Force dist | Out-Null
go build -ldflags '-s -w -H=windowsgui -extldflags=-static' -o dist/CodexCommandTrace.exe .
```

`ci` 标签用于无图形显示环境下的测试；实际桌面构建不使用该标签。Windows 构建静态链接编译器运行库，仍依赖系统自带的窗口和 OpenGL 组件。

## 日志监听与命令配对

默认监听 `$CODEX_HOME/sessions`，未设置时使用用户目录下 `.codex/sessions`。默认从现有文件末尾开始，新文件从头读取；`-history` 可读取已有历史。目录事件触发增量读取，每 3 秒扫描补查，支持新建子目录、文件截短或替换，以及跨写入的半行 JSON。

支持 `exec_command`、`shell_command`、`shell`、`exec` 和 `write_stdin`。通过日志文件和调用 ID 关联命令与结果；后台进程通过返回的进程会话编号关联后续输出。桌面版 `exec` 中的常见字符串参数可直接提取，不执行包装代码；动态参数暂无法还原。

连续 `text(await tools.NAME(...))` 顺序调用逐条配对，中间非命令工具的返回也计入位置，但不展示。仅在完整输出数量与调用数量一致时建立对应；并行、条件分支、额外输出或缺失返回等无法确认的结构退回整组展示并提示。

会话名称来自 sessions 同级 `session_index.jsonl` 的 `thread_name`。详情还展示日志中可确认的会话 ID、工作目录、客户端版本、模型提供方、Git 信息、日志文件、调用 ID、时间及执行结果元数据。

监听与解析在生产端串行处理，产生结构化 `commandEvent`，通过容量为 256 的 `chan` 交给 GUI 或控制台消费者。GUI 使用 `fyne.Do` 调度到界面线程，窗口关闭会取消监听和后台动画。

## 界面与演示

主窗口固定大小，宽度约为屏幕四分之一，高度为宽度的 16/9，必要时按屏幕高度缩小。卡片随内容增长，长输出显示中间五行及上下渐隐上下文。每约 15 毫秒推进命令打字，每约 75 毫秒追加一行输出。

卡片最多排版最近 16 行及约 8 KB 尾部文字，完整已接收结果保留在详情中。历史上限为 20 条，清理旧卡片时同步清理动画和待返回结果引用。

独立窗口从主窗口所在显示器开始，自上而下、从左到右排布，再使用其他显示器，避开主窗口与任务栏；支持不同缩放比例、负坐标和镜像屏幕去重。开启前已有卡片不转换；关闭时逐个淡出，保留卡片恢复到主窗口。

演示数据和发送节奏位于 `gui_demo.go`，不实际执行命令，也不监听日志：

1. `demoExamples()` 的原有 5 条固定命令，每条等待 0.8 秒出现，再等待 0.6 秒返回结果。
2. `pwsh` 等待命令在出现 5 秒后返回 `wait complete`，留出开启独立窗口的时间。
3. `demoRecordingExamples()` 的 15 条固定开发场景，以随机 0.5～2 秒的命令开始间隔播放，模拟结果延迟为 150～450 毫秒，最后一条输出“任务完成”。

演示仅播放一次；时间戳使用实际时间，展示耗时为模拟文字。subtitle 仍显示配置的日志目录。

## Windows 自动打包与发布

工作流为 `.github/workflows/package-windows.yml`，仅在推送 tag 时触发。普通分支推送不触发。发布新版本，例如：

```powershell
git tag -a v0.1.1 -m 'Release v0.1.1'
git push origin v0.1.1
```

工作流在 Windows runner 上使用 Go 和 MSYS2 UCRT64 GCC，依次验证依赖、运行测试和静态检查、构建 Windows amd64 程序并生成 ZIP。包内包含 `CodexCommandTrace.exe`、`README.md` 和程序的 `SHA256SUMS.txt`。

打包成功后自动创建对应 tag 的 GitHub Release，生成发布说明并上传 ZIP 附件；已有 Release 时重新上传同名附件。工作流使用内置 `GITHUB_TOKEN`，需要 `contents: write` 权限，无需另设个人令牌。打包失败不会执行发布步骤。

[Actions](https://github.com/dongfg/codex-cmd-trace/actions) 中也保留 ZIP artifact 30 天；[Releases](https://github.com/dongfg/codex-cmd-trace/releases) 提供面向用户的下载入口。旧 tag 使用其所指提交中的工作流；修改工作流后不会自动为旧版本补发 Release。
