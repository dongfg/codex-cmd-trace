package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestWrappedCommands(t *testing.T) {
	input := `text(await tools.apply_patch("tools.exec_command({cmd:'fake'})"));
 text(await tools.exec_command({cmd:"go test ./...", "workdir":"E:\\Projects",max_output_tokens:1000}));
 text(await tools.exec_command({cmd:'git status',workdir:'E:\\Projects'}));`
	got := describeCall("exec", input)
	for _, want := range []string{"命令: go test ./...", "命令: git status", `目录: E:\Projects`} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q: %s", want, got)
		}
	}
	if strings.Contains(got, "fake") || strings.Contains(got, "max_output_tokens") {
		t.Fatal(got)
	}
	if describeCall("exec", `text(await tools.apply_patch("patch"));`) != "" {
		t.Fatal("non-command wrapper shown")
	}
}

func TestStructuredOutput(t *testing.T) {
	raw := json.RawMessage(`[{"type":"input_text","text":"Script completed\nWall time 1 seconds\nOutput:\n"},{"type":"input_text","text":"{\"exit_code\":1,\"wall_time_seconds\":0.5,\"output\":\"failed\\n\"}"}]`)
	got := readableOutput(raw)
	if !strings.Contains(got, "失败（退出码 1）") || !strings.Contains(got, "failed") || strings.Contains(got, "input_text") {
		t.Fatal(got)
	}
	got = cleanOutput(`{"session_id":123,"output":"","wall_time_seconds":1}`)
	if !strings.Contains(got, "执行中") || !strings.Contains(got, "123") {
		t.Fatal(got)
	}
}
