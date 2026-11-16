package main

import "encoding/json"

func executionFromText(text string) executionInfo {
	var info executionInfo
	_ = json.Unmarshal([]byte(text), &info)
	return info
}
func executionFromOutput(raw json.RawMessage) executionInfo {
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return executionFromText(text)
	}
	var blocks []struct {
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &blocks) == nil {
		var found []executionInfo
		for _, b := range blocks {
			info := executionFromText(b.Text)
			if info.ExitCode != nil || info.ProcessSession != nil || info.ChunkID != "" {
				found = append(found, info)
			}
		}
		if len(found) == 1 {
			return found[0]
		}
		return executionInfo{}
	}
	return executionFromText(string(raw))
}
