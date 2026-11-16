package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIncrementalAndTruncatedLog(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	write := func(s string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(s), 0600); err != nil {
			t.Fatal(err)
		}
	}
	tr := &tracer{files: map[string]*cursor{}, calls: map[string]bool{}}
	write("old record\n")
	if err := tr.read(path, true); err != nil {
		t.Fatal(err)
	}
	part := `{"type":"response_item","payload":{"type":"function_call","name":"exec_command","call_id":"one","arguments":"{\"cmd\":\"echo hello\"}"}}`
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.WriteString(part); err != nil {
		t.Fatal(err)
	}
	f.Close()
	if err := tr.read(path, false); err != nil {
		t.Fatal(err)
	}
	if len(tr.calls) != 0 {
		t.Fatal("partial line was parsed")
	}
	f, err = os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("\n")
	f.Close()
	if err := tr.read(path, false); err != nil {
		t.Fatal(err)
	}
	if !tr.calls[path+":one"] {
		t.Fatal("completed line was not parsed")
	}
	if err := tr.read(path, false); err != nil {
		t.Fatal(err)
	}
	if len(tr.calls) != 1 {
		t.Fatal("unexpected duplicate")
	}
	write("{}\n")
	if err := tr.read(path, false); err != nil {
		t.Fatal(err)
	}
	if tr.files[path].offset != 3 || len(tr.files[path].pending) != 0 {
		t.Fatal("truncation did not reset cursor")
	}
	tr.line(path, []byte(`{"type":"response_item","payload":{"type":"function_call_output","call_id":"one","output":"done"}}`))
	if len(tr.calls) != 0 {
		t.Fatal("result was not correlated")
	}
}
