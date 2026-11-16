package main

import "strings"

func demoExamples() []commandResult {
	return []commandResult{
		{Description: "命令: go version", Output: "成功 · 0.02 秒\ngo version go1.27.4 windows/amd64"},
		{Description: "命令: git status --short", Output: "成功 · 0.03 秒\n工作区干净"},
		{Description: "命令: go test ./...", Output: "成功 · 0.52 秒\nok  codex-cmd-trace\nok  session-parser"},
		{Description: "命令: go build .", Output: "成功 · 1.28 秒\n" + strings.Join([]string{"Resolving modules…", "Loading configuration…", "Building watcher…", "Building parser…", "Building event queue…", "Building widgets…", "Loading fonts…", "Linking graphics…", "Linking application…", "Writing executable…", "Build completed."}, "\n")},
		{Description: "命令: go test -run TestConnection ./...", Output: "失败（退出码 1） · 0.21 秒\n--- FAIL: TestConnection\n    connection timeout\nFAIL"},
	}
}
