package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Session titles are optional metadata; missing or damaged indexes never stop tracing.
func sessionTitleLookup(root string) func(string) string {
	index := filepath.Join(filepath.Dir(root), "session_index.jsonl")
	titles := map[string]string{}
	var checked time.Time
	var modified time.Time
	var size int64 = -1
	return func(path string) string {
		if time.Since(checked) > 3*time.Second {
			checked = time.Now()
			if info, err := os.Stat(index); err == nil && (info.ModTime() != modified || info.Size() != size) {
				if data, err := os.ReadFile(index); err == nil {
					next := map[string]string{}
					for _, line := range bytes.Split(data, []byte{'\n'}) {
						var r struct {
							ID    string `json:"id"`
							Title string `json:"thread_name"`
						}
						if json.Unmarshal(line, &r) == nil && r.ID != "" && r.Title != "" {
							next[r.ID] = r.Title
						}
					}
					titles = next
					modified = info.ModTime()
					size = info.Size()
				}
			}
		}
		name := strings.TrimSuffix(filepath.Base(path), ".jsonl")
		if len(name) >= 36 {
			return titles[name[len(name)-36:]]
		}
		return ""
	}
}
