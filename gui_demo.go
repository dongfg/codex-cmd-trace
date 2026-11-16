package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"
)

func demoExamples() []commandResult {
	return []commandResult{
		{Description: "命令: go version", Output: "成功 · 0.02 秒\ngo version go1.27.4 windows/amd64"},
		{Description: "命令: git status --short", Output: "成功 · 0.03 秒\n工作区干净"},
		{Description: "命令: go test ./...", Output: "成功 · 0.52 秒\nok  codex-cmd-trace\nok  session-parser"},
		{Description: "命令: go build .", Output: "成功 · 1.28 秒\n" + strings.Join([]string{"Resolving modules…", "Loading configuration…", "Building watcher…", "Building parser…", "Building event queue…", "Building widgets…", "Loading fonts…", "Linking graphics…", "Linking application…", "Writing executable…", "Build completed."}, "\n")},
		{Description: "命令: go test -run TestConnection ./...", Output: "失败（退出码 1） · 0.21 秒\n--- FAIL: TestConnection\n    connection timeout\nFAIL"},
	}
}

func demoRecordingExamples() []commandResult {
	return []commandResult{
		{Description: "命令: git branch --show-current", Output: "成功 · 0.03 秒\nmaster"},
		{Description: "命令: rg --files -g '*.go'", Output: "成功 · 0.04 秒\nmain.go\nsessions.go\nsequence.go\ngui.go\ngui_demo.go\ngui_floating.go\nfloating_layout.go\nwindow_windows.go"},
		{Description: "命令: go env GOOS GOARCH CGO_ENABLED", Output: "成功 · 0.02 秒\nwindows\namd64\n1"},
		{Description: "命令: git diff --stat", Output: "成功 · 0.03 秒\n gui_demo.go | 84 ++++++++++++++++++++++++++++++++++++++++++++\n 1 file changed, 84 insertions(+)"},
		{Description: "命令: go mod verify", Output: "成功 · 0.18 秒\nall modules verified"},
		{Description: "命令: go test -run TestSequence -v ./...", Output: "成功 · 0.24 秒\n=== RUN   TestSequence\n--- PASS: TestSequence (0.00s)\nPASS\nok  github.com/dongfg/codex-cmd-trace  0.024s"},
		{Description: "命令: go test -run TestFloating -v ./...", Output: "成功 · 0.31 秒\n=== RUN   TestFloatingLayout\n--- PASS: TestFloatingLayout (0.00s)\n=== RUN   TestFloatingEviction\n--- PASS: TestFloatingEviction (0.00s)\nPASS\nok  github.com/dongfg/codex-cmd-trace  0.031s"},
		{Description: "命令: go vet ./...", Output: "成功 · 0.27 秒"},
		{Description: "命令: go test -count=1 ./...", Output: "成功 · 0.42 秒\nok  github.com/dongfg/codex-cmd-trace  0.312s"},
		{Description: "命令: pwsh -NoProfile -Command \"New-Item -ItemType Directory -Force dist | Out-Null\"", Output: "成功 · 0.21 秒"},
		{Description: "命令: go build -ldflags '-s -w -H=windowsgui' -o dist/CodexCommandTrace.exe .", Output: "成功 · 0.38 秒"},
		{Description: "命令: pwsh -NoProfile -Command \"Get-Item dist/CodexCommandTrace.exe | Select-Object Name, Length\"", Output: "成功 · 0.19 秒\nName                       Length\n----                       ------\nCodexCommandTrace.exe     28475392"},
		{Description: "命令: pwsh -NoProfile -Command \"Compress-Archive -Path dist/CodexCommandTrace.exe -DestinationPath dist/windows-amd64.zip -Force\"", Output: "成功 · 0.35 秒"},
		{Description: "命令: git status --short", Output: "成功 · 0.02 秒\n M gui_demo.go"},
		{Description: "命令: pwsh -NoProfile -Command \"Write-Output '任务完成'\"", Output: "成功 · 0.20 秒\n任务完成"},
	}
}

func demoFeed(ctx context.Context, send func(commandEvent)) {
	wait := func(delay time.Duration) bool {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return false
		case <-timer.C:
			return true
		}
	}
	index := 0
	play := func(r commandResult, duration time.Duration) bool {
		index++
		e := commandEvent{Kind: "started", Key: fmt.Sprint(index), ID: fmt.Sprintf("demo-%02d", index), Time: time.Now().Format("15:04:05"), Description: r.Description, Commands: []string{r.Description}, SessionTitle: "演示"}
		send(e)
		if !wait(duration) {
			return false
		}
		e.Kind, e.Paired, e.Results = "result", true, []commandResult{r}
		send(e)
		return true
	}
	for _, r := range demoExamples() {
		if !wait(800*time.Millisecond) || !play(r, 600*time.Millisecond) {
			return
		}
	}
	gate := commandResult{Description: `命令: pwsh -NoProfile -Command "Start-Sleep -Seconds 5; Write-Output 'wait complete'"`, Output: "成功 · 5.00 秒\nwait complete"}
	if !wait(800*time.Millisecond) || !play(gate, 5*time.Second) {
		return
	}
	previousDuration := time.Duration(0)
	for _, r := range demoRecordingExamples() {
		// Keep command start times 0.5–2 seconds apart, including result latency.
		interval := 500*time.Millisecond + time.Duration(rand.IntN(1501))*time.Millisecond
		duration := time.Duration(150+rand.IntN(301)) * time.Millisecond
		if !wait(interval-previousDuration) || !play(r, duration) {
			return
		}
		previousDuration = duration
	}
}
