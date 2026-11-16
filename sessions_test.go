package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSessionTitleLookup(t *testing.T) {
	root := filepath.Join(t.TempDir(), "sessions")
	id := "01a1e23a-ac6b-7211-b4f3-45fe3277c9cd"
	index := filepath.Join(filepath.Dir(root), "session_index.jsonl")
	data := "invalid\n{\"id\":\"" + id + "\",\"thread_name\":\"旧名称\"}\n{\"id\":\"" + id + "\",\"thread_name\":\"命令监听\"}\n"
	if err := os.WriteFile(index, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	lookup := sessionTitleLookup(root)
	if got := lookup(filepath.Join(root, "rollout-2026-11-16T10-40-58-"+id+".jsonl")); got != "命令监听" {
		t.Fatal(got)
	}
	if got := lookup(filepath.Join(root, "unknown.jsonl")); got != "" {
		t.Fatal(got)
	}
}
