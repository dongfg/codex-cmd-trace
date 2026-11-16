package main

import (
	"encoding/json"
	"strings"
)

// Each slot is one text(await tools.NAME(...)) emission, including hidden tools.
// Only straight-line statements are accepted; unknown control flow never gets paired.
func sequencePlan(input string) ([]string, bool) {
	ts := jsTokens(input)
	var slots []string
	for i := 0; i < len(ts); {
		if ts[i].value == ";" && !ts[i].literal {
			i++
			continue
		}
		start := i
		stack := []string{}
		for i < len(ts) {
			tok := ts[i]
			if !tok.literal {
				switch tok.value {
				case "(", "{", "[":
					stack = append(stack, tok.value)
				case ")", "}", "]":
					if len(stack) == 0 {
						return nil, false
					}
					stack = stack[:len(stack)-1]
				case ";":
					if len(stack) == 0 {
						goto statement
					}
				}
			}
			i++
		}
	statement:
		s := ts[start:i]
		if i < len(ts) {
			i++
		}
		if len(s) == 0 {
			continue
		}
		if len(s) >= 10 && tokensAre(s[:6], "text", "(", "await", "tools", ".", s[5].value) && tokensAre(s[6:7], "(") && tokensAre(s[len(s)-2:], ")", ")") {
			name := s[5].value
			// There must be exactly one tool invocation and no nested emitters/control flow.
			for _, t := range s[7 : len(s)-2] {
				if !t.literal && (t.value == "await" || t.value == "text" || t.value == "tools" || t.value == "Promise") {
					return nil, false
				}
			}
			detail := ""
			if isCommand(name) {
				detail = describeCall("exec", "text(await tools."+name+"("+tokensArgs(s[7:len(s)-2])+"));")
				if detail == "" {
					detail = "命令参数为动态表达式，暂无法提取"
				}
			}
			slots = append(slots, detail)
			continue
		}
		// Local constants (e.g. a script string) emit no output.
		if s[0].value != "const" && s[0].value != "let" && s[0].value != "var" {
			return nil, false
		}
		for _, t := range s {
			if !t.literal && (t.value == "tools" || t.value == "text" || t.value == "await" || t.value == "Promise") {
				return nil, false
			}
		}
	}
	return slots, len(slots) > 0
}

func tokensAre(ts []jsToken, values ...string) bool {
	if len(ts) != len(values) {
		return false
	}
	for i, t := range ts {
		if t.literal || t.value != values[i] {
			return false
		}
	}
	return true
}

func isCommand(name string) bool {
	return name == "exec_command" || name == "shell_command" || name == "shell" || name == "write_stdin"
}

// Reuse the existing display parser for object arguments, preserving string values.
func tokensArgs(ts []jsToken) string {
	var b strings.Builder
	for _, t := range ts {
		if t.literal {
			v, _ := json.Marshal(t.value)
			b.Write(v)
		} else {
			b.WriteString(t.value)
		}
		b.WriteByte(' ')
	}
	// describeCall(exec) understands JS object keys and literal arguments.
	return b.String()
}

func sequenceResults(slots []string, raw json.RawMessage) ([]commandResult, bool) {
	if len(slots) == 0 {
		return nil, false
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &blocks) != nil {
		return nil, false
	}
	var outputs []string
	for i, b := range blocks {
		if i == 0 && b.Type == "input_text" && strings.HasPrefix(b.Text, "Script completed\n") {
			continue
		}
		if b.Type != "input_text" {
			return nil, false
		}
		outputs = append(outputs, b.Text)
	}
	if len(outputs) != len(slots) {
		return nil, false
	}
	var paired []commandResult
	for i, detail := range slots {
		if detail == "" {
			continue
		}
		out := cleanOutput(outputs[i])
		if out == "" {
			out = "（无输出）"
		}
		paired = append(paired, commandResult{Description: detail, Output: out, Execution: executionFromText(outputs[i])})
	}
	return paired, true
}
