package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

func shortID(s string) string {
	s = strings.TrimPrefix(s, "call_")
	if len(s) > 8 {
		return s[:8]
	}
	return s
}

// Tokenize without evaluating JavaScript. String contents never become code tokens.
type jsToken struct {
	value   string
	literal bool
}

func jsTokens(s string) []jsToken {
	var out []jsToken
	for i := 0; i < len(s); {
		c := s[i]
		if unicode.IsSpace(rune(c)) {
			i++
			continue
		}
		if i+1 < len(s) && s[i:i+2] == "//" {
			for i < len(s) && s[i] != '\n' {
				i++
			}
			continue
		}
		if i+1 < len(s) && s[i:i+2] == "/*" {
			j := strings.Index(s[i+2:], "*/")
			if j < 0 {
				break
			}
			i += j + 4
			continue
		}
		if c == '"' || c == '\'' || c == '`' {
			quote := c
			start := i
			i++
			for i < len(s) {
				if s[i] == '\\' {
					i += 2
					continue
				}
				if s[i] == quote {
					break
				}
				i++
			}
			if i >= len(s) {
				break
			}
			raw := s[start : i+1]
			i++
			var v string
			if quote == '"' {
				if json.Unmarshal([]byte(raw), &v) != nil {
					continue
				}
			} else {
				body := raw[1 : len(raw)-1]
				if quote == '`' && strings.Contains(body, "${") {
					out = append(out, jsToken{"<动态模板>", true})
					continue
				}
				body = strings.ReplaceAll(body, "\\'", "'")
				body = strings.ReplaceAll(body, "\\`", "`")
				body = strings.ReplaceAll(body, "\"", "\\\"")
				body = strings.ReplaceAll(body, "\n", "\\n")
				body = strings.ReplaceAll(body, "\r", "\\r")
				if json.Unmarshal([]byte("\""+body+"\""), &v) != nil {
					v = body
				}
			}
			out = append(out, jsToken{v, true})
			continue
		}
		if c == '_' || c == '$' || unicode.IsLetter(rune(c)) || c >= '0' && c <= '9' {
			j := i + 1
			for j < len(s) && (s[j] == '_' || s[j] == '$' || unicode.IsLetter(rune(s[j])) || s[j] >= '0' && s[j] <= '9') {
				j++
			}
			out = append(out, jsToken{s[i:j], false})
			i = j
			continue
		}
		out = append(out, jsToken{string(c), false})
		i++
	}
	return out
}

func describeCall(name, input string) string {
	if name == "exec" {
		ts := jsTokens(input)
		var commands []string
		for i := 0; i+4 < len(ts); i++ {
			if ts[i].literal || ts[i].value != "tools" || ts[i+1].value != "." || ts[i+3].value != "(" || ts[i+4].value != "{" {
				continue
			}
			tool := ts[i+2].value
			if tool != "exec_command" && tool != "shell_command" && tool != "shell" && tool != "write_stdin" {
				continue
			}
			depth := 1
			args := map[string]any{}
			for j := i + 5; j < len(ts) && depth > 0; j++ {
				if ts[j].literal {
					if depth == 1 && j+2 < len(ts) && ts[j+1].value == ":" && ts[j+2].literal {
						args[ts[j].value] = ts[j+2].value
					}
					continue
				}
				if ts[j].value == "{" {
					depth++
				}
				if ts[j].value == "}" {
					depth--
					continue
				}
				if depth == 1 && j+2 < len(ts) && ts[j+1].value == ":" {
					v := ts[j+2]
					if v.literal {
						args[ts[j].value] = v.value
					} else if n, err := strconv.Atoi(v.value); err == nil {
						args[ts[j].value] = n
					}
				}
			}
			// Quoted object keys are also valid JavaScript.
			if text := describeArgs(tool, args); text != "" {
				commands = append(commands, text)
			} else {
				commands = append(commands, "命令参数为动态表达式，暂无法提取")
			}
		}
		if len(commands) == 0 {
			return ""
		}
		return strings.Join(commands, "\n\n")
	}
	var args map[string]any
	if json.Unmarshal([]byte(input), &args) != nil {
		return "命令: " + input
	}
	return describeArgs(name, args)
}

func describeArgs(name string, args map[string]any) string {
	if name == "write_stdin" {
		action := "等待后续输出"
		if s, _ := args["chars"].(string); s != "" {
			action = fmt.Sprintf("输入: %q", s)
		}
		return fmt.Sprintf("%s（进程会话 %v）", action, args["session_id"])
	}
	cmd, _ := args["cmd"].(string)
	if cmd == "" {
		cmd, _ = args["command"].(string)
	}
	if a, ok := args["command"].([]any); ok {
		parts := []string{}
		for _, v := range a {
			parts = append(parts, fmt.Sprint(v))
		}
		cmd = strings.Join(parts, " ")
	}
	if cmd == "" {
		return ""
	}
	out := "命令: " + cmd
	if dir, _ := args["workdir"].(string); dir != "" {
		out = "目录: " + dir + "\n" + out
	}
	return out
}

func readableOutput(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return cleanOutput(s)
	}
	var blocks []struct {
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &blocks) == nil {
		var parts []string
		for _, b := range blocks {
			if strings.HasPrefix(b.Text, "Script completed\n") || strings.HasPrefix(b.Text, "Script running with cell ID") {
				continue
			}
			v := cleanOutput(b.Text)
			if v != "" {
				parts = append(parts, v)
			}
		}
		if len(parts) == 0 {
			return "（没有命令输出）"
		}
		return strings.Join(parts, "\n\n")
	}
	return cleanOutput(string(raw))
}

func cleanOutput(s string) string {
	var r struct {
		Output    *string `json:"output"`
		ExitCode  *int    `json:"exit_code"`
		SessionID *int    `json:"session_id"`
		Wall      float64 `json:"wall_time_seconds"`
	}
	if json.Unmarshal([]byte(s), &r) == nil {
		if r.Output == nil {
			return ""
		}
		status := "执行中"
		if r.ExitCode != nil {
			if *r.ExitCode == 0 {
				status = "成功"
			} else {
				status = fmt.Sprintf("失败（退出码 %d）", *r.ExitCode)
			}
		}
		if r.SessionID != nil {
			status += fmt.Sprintf(" · 进程会话 %d", *r.SessionID)
		}
		out := strings.TrimRight(*r.Output, "\r\n")
		if out == "" {
			out = "（无输出）"
		}
		return fmt.Sprintf("%s · %.2f 秒\n%s", status, r.Wall, out)
	}
	// CLI versions store a textual envelope rather than a JSON object.
	if at := strings.Index(s, "\nFinal output:\n"); at >= 0 {
		return s[at+len("\nFinal output:\n"):]
	}
	return s
}
