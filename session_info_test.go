package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSessionMetadataForDetails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rollout-01a1e23a-ac6b-7211-b4f3-45fe3277c9cd.jsonl")
	log := `{"type":"session_meta","payload":{"id":"session-id","cwd":"E:/Project","cli_version":"1.2.3","originator":"desktop","model_provider":"openai","source":"cli","git":{"branch":"main","commit_hash":"abc123"},"base_instructions":"PRIVATE PROMPT"}}` + "\n"
	if err := os.WriteFile(path, []byte(log), 0600); err != nil {
		t.Fatal(err)
	}
	info := sessionInfoLookup()(path)
	c := &traceCard{metadata: commandEvent{Session: info, SessionTitle: "命令监听", CallID: "full-call-id", SourceFile: path, ToolName: "exec", StartedAt: "2026-11-16T05:00:00Z"}, commandIndex: 2, commandCount: 3}
	value := cardInfoText(c)
	for _, want := range []string{"session-id", "命令监听", "E:/Project", "1.2.3", "full-call-id", path, "main", "2 / 3"} {
		if !strings.Contains(value, want) {
			t.Fatal("missing metadata", want)
		}
	}
	if strings.Contains(value, "PRIVATE PROMPT") {
		t.Fatal("unrelated prompt metadata exposed")
	}
	if got := sessionIDFromPath(path); got != "01a1e23a-ac6b-7211-b4f3-45fe3277c9cd" {
		t.Fatal(got)
	}
}
