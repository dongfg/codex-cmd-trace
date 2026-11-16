package main

import (
	"encoding/json"
	"testing"
)

func TestExecutionMetadataAvoidsAmbiguousBatch(t *testing.T) {
	one := json.RawMessage(`[{"text":"Script completed\n"},{"text":"{\"exit_code\":0,\"wall_time_seconds\":0.25,\"session_id\":42}"}]`)
	info := executionFromOutput(one)
	if info.ExitCode == nil || *info.ExitCode != 0 || info.WallTime == nil || *info.WallTime != 0.25 || info.ProcessSession == nil || *info.ProcessSession != 42 {
		t.Fatal(info)
	}
	batch := json.RawMessage(`[{"text":"{\"exit_code\":0}"},{"text":"{\"exit_code\":1}"}]`)
	if executionFromOutput(batch).ExitCode != nil {
		t.Fatal("ambiguous metadata assigned to one command")
	}
}
