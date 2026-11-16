package main

import (
	"fmt"
	"io"
)

// commandEvent is independent of terminal formatting and can be consumed by a UI.
type commandEvent struct {
	Kind         string
	ID           string
	Time         string
	Description  string
	Commands     []string
	Key          string
	SessionTitle string
	Output       string
	Paired       bool
	Results      []commandResult
	CallID       string
	ToolName     string
	SourceFile   string
	StartedAt    string
	CompletedAt  string
	RawInput     string
	Session      sessionInfo
}

type commandResult struct {
	Description string
	Output      string
	Execution   executionInfo
}

type executionInfo struct {
	ExitCode       *int     `json:"exit_code"`
	WallTime       *float64 `json:"wall_time_seconds"`
	ProcessSession *int     `json:"session_id"`
	ChunkID        string   `json:"chunk_id"`
}

func renderEvent(w io.Writer, e commandEvent) {
	if e.Kind == "started" {
		fmt.Fprintf(w, "\n━━ %s · %s ━━\n%s\n", e.Time, e.ID, e.Description)
		return
	}
	fmt.Fprintf(w, "\n── 结果 · %s ──\n", e.ID)
	if e.Paired {
		for i, r := range e.Results {
			fmt.Fprintf(w, "\n[%d] %s\n%s\n", i+1, r.Description, r.Output)
		}
		return
	}
	fmt.Fprintln(w, e.Output)
}
