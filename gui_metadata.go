package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func recordTime(value string) string {
	if t, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return t.Local().Format("2006-01-02 15:04:05.000 -07:00")
	}
	return value
}
func cardInfoText(c *traceCard) string {
	m := c.metadata
	s := m.Session
	var rows []string
	add := func(label, value string) {
		if value != "" {
			rows = append(rows, label+": "+value)
		}
	}
	title := m.SessionTitle
	if title == "" {
		title = "日志未提供名称"
	}
	add("会话名称", title)
	add("会话 ID", s.ID)
	add("会话工作目录", s.CWD)
	add("会话创建", recordTime(s.Timestamp))
	add("日志文件", m.SourceFile)
	add("客户端", s.Originator)
	add("CLI 版本", s.CLIVersion)
	add("模型提供方", s.ModelProvider)
	if len(s.Source) > 0 {
		var source string
		if json.Unmarshal(s.Source, &source) != nil {
			source = string(s.Source)
		}
		add("会话来源", source)
	}
	add("Git 分支", s.Git.Branch)
	add("Git 提交", s.Git.Commit)
	add("Git 仓库", s.Git.Repository)
	add("工具", m.ToolName)
	id := m.CallID
	if id == "" {
		id = m.ID
	}
	add("调用 ID", id)
	if c.commandCount > 0 {
		add("组内命令", fmt.Sprintf("%d / %d", c.commandIndex, c.commandCount))
	}
	add("调用记录时间", recordTime(m.StartedAt))
	add("返回记录时间", recordTime(m.CompletedAt))
	if c.execution.ExitCode != nil {
		add("退出码", fmt.Sprint(*c.execution.ExitCode))
	}
	if c.execution.WallTime != nil {
		add("执行耗时", fmt.Sprintf("%.3f 秒", *c.execution.WallTime))
	}
	if c.execution.ProcessSession != nil {
		add("后台进程会话", fmt.Sprint(*c.execution.ProcessSession))
	}
	add("输出块 ID", c.execution.ChunkID)
	if c.finished {
		if m.Paired {
			add("关联方式", "命令与结果已逐条配对")
		} else {
			add("关联方式", "整组输出，未确认逐条对应")
		}
	}
	return strings.Join(rows, "\n")
}
