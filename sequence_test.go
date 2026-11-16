package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestSequenceWithHiddenTool(t *testing.T) {
	input := `text(await tools.exec_command({cmd:"echo first"}));
text(await tools.apply_patch("patch with tools.exec_command({cmd:'fake'})"));
text(await tools.exec_command({cmd:"echo second"}));`
	slots, ok := sequencePlan(input)
	if !ok || len(slots) != 3 || slots[1] != "" {
		t.Fatalf("bad plan: %#v %v", slots, ok)
	}
	raw := json.RawMessage(`[{"type":"input_text","text":"Script completed\nWall time 1 seconds\nOutput:\n"},{"type":"input_text","text":"{\"exit_code\":0,\"output\":\"first\"}"},{"type":"input_text","text":"{}"},{"type":"input_text","text":"{\"exit_code\":1,\"output\":\"second\"}"}]`)
	results, ok := sequenceResults(slots, raw)
	if !ok || len(results) != 2 {
		t.Fatalf("bad results: %#v", results)
	}
	if results[0].Description != "命令: echo first" || !strings.HasSuffix(results[0].Output, "\nfirst") {
		t.Fatal(results[0])
	}
	if results[1].Description != "命令: echo second" || !strings.Contains(results[1].Output, "失败") || !strings.HasSuffix(results[1].Output, "\nsecond") {
		t.Fatal(results[1])
	}
	var events []commandEvent
	tr := &tracer{calls: map[string]bool{}, emit: func(e commandEvent) { events = append(events, e) }}
	call, _ := json.Marshal(map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call", "name": "exec", "call_id": "test", "input": input}})
	tr.line("file", call)
	output, _ := json.Marshal(map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call_output", "call_id": "test", "output": raw}})
	tr.line("file", output)
	if len(events) != 2 || !events[1].Paired || len(events[1].Results) != 2 {
		t.Fatalf("bad events: %#v", events)
	}
	var b bytes.Buffer
	renderEvent(&b, events[1])
	s := b.String()
	if strings.Index(s, "\nfirst") > strings.Index(s, "命令: echo second") {
		t.Fatal("output was grouped instead of paired", s)
	}
	if len(tr.pending) != 0 || len(tr.plans) != 0 {
		t.Fatal("pending state leaked")
	}
	if _, ok := sequenceResults(slots, json.RawMessage(`[{"type":"input_text","text":"only one"}]`)); ok {
		t.Fatal("missing outputs were paired")
	}
}

func TestSequenceRejectsUnknownControlFlow(t *testing.T) {
	for _, s := range []string{`await Promise.all([tools.exec_command({cmd:"a"}),tools.exec_command({cmd:"b"})]);`, `if (ok) {text(await tools.exec_command({cmd:"a"}));}`, `text("extra");text(await tools.exec_command({cmd:"a"}));`} {
		if _, ok := sequencePlan(s); ok {
			t.Fatal("unsafe plan", s)
		}
	}
	slots, ok := sequencePlan("const s=`script`; text(await tools.exec_command({cmd:`${s}`}));text(await tools.exec_command({cmd:'b'}));")
	if !ok || len(slots) != 2 {
		t.Fatalf("script prefix unsupported: %#v", slots)
	}
}
