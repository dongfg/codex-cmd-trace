package main

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type sessionInfo struct {
	ID            string          `json:"id"`
	CWD           string          `json:"cwd"`
	Timestamp     string          `json:"timestamp"`
	Originator    string          `json:"originator"`
	CLIVersion    string          `json:"cli_version"`
	ModelProvider string          `json:"model_provider"`
	Source        json.RawMessage `json:"source"`
	Git           struct {
		Branch     string `json:"branch"`
		Commit     string `json:"commit_hash"`
		Repository string `json:"repository_url"`
	} `json:"git"`
}

var sessionUUID = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func sessionIDFromPath(path string) string {
	return sessionUUID.FindString(strings.TrimSuffix(filepath.Base(path), ".jsonl"))
}

func sessionInfoLookup() func(string) sessionInfo {
	cache := map[string]sessionInfo{}
	return func(path string) sessionInfo {
		if info, ok := cache[path]; ok {
			return info
		}
		info := sessionInfo{ID: sessionIDFromPath(path)}
		f, err := os.Open(path)
		if err != nil {
			return info
		}
		defer f.Close()
		scanner := bufio.NewScanner(io.LimitReader(f, 2<<20))
		scanner.Buffer(make([]byte, 64<<10), 2<<20)
		for n := 0; n < 3 && scanner.Scan(); n++ {
			var row struct {
				Type    string      `json:"type"`
				Payload sessionInfo `json:"payload"`
			}
			if json.Unmarshal(scanner.Bytes(), &row) == nil && row.Type == "session_meta" {
				info = row.Payload
				if info.ID == "" {
					info.ID = sessionIDFromPath(path)
				}
				cache[path] = info
				break
			}
		}
		return info
	}
}
