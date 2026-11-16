package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"io"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

type cursor struct {
	info    os.FileInfo
	offset  int64
	pending []byte
}
type tracer struct {
	files        map[string]*cursor
	calls        map[string]bool
	history      bool
	plans        map[string][]string
	pending      map[string]commandEvent
	emit         func(commandEvent)
	sessionTitle func(string) string
	sessionInfo  func(string) sessionInfo
}

func (t *tracer) read(path string, initial bool) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	c := t.files[path]
	if c == nil {
		c = &cursor{info: info}
		t.files[path] = c
		if initial && !t.history {
			c.offset = info.Size()
			return nil
		}
	}
	if !os.SameFile(c.info, info) || info.Size() < c.offset {
		c.offset = 0
		c.pending = nil
	}
	c.info = info
	if _, err = f.Seek(c.offset, io.SeekStart); err != nil {
		return err
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return err
	}
	c.offset += int64(len(data))
	c.pending = append(c.pending, data...)
	for {
		n := bytes.IndexByte(c.pending, '\n')
		if n < 0 {
			break
		}
		t.line(path, c.pending[:n])
		c.pending = c.pending[n+1:]
	}
	if len(c.pending) == 0 {
		c.pending = nil
	}
	return nil
}

func (t *tracer) line(path string, data []byte) {
	var r struct {
		Timestamp string `json:"timestamp"`
		Type      string `json:"type"`
		Payload   struct {
			Type      string          `json:"type"`
			Name      string          `json:"name"`
			CallID    string          `json:"call_id"`
			Arguments string          `json:"arguments"`
			Input     string          `json:"input"`
			Output    json.RawMessage `json:"output"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(data, &r); err != nil {
		log.Printf("跳过无效 JSONL %s: %v", path, err)
		return
	}
	if r.Type != "response_item" {
		return
	}
	p := r.Payload
	key := path + ":" + p.CallID
	switch p.Type {
	case "function_call", "custom_tool_call":
		if p.Name != "exec_command" && p.Name != "shell_command" && p.Name != "shell" && p.Name != "exec" && p.Name != "write_stdin" {
			return
		}
		if t.calls[key] {
			return
		}
		input := p.Arguments
		if input == "" {
			input = p.Input
		}
		detail := describeCall(p.Name, input)
		if detail == "" {
			return
		}
		t.calls[key] = true
		if p.Name == "exec" {
			if t.plans == nil {
				t.plans = map[string][]string{}
			}
			t.plans[key] = nil
			if slots, ok := sequencePlan(input); ok {
				if t.plans == nil {
					t.plans = map[string][]string{}
				}
				t.plans[key] = slots
			}
		}
		stamp := r.Timestamp
		if tm, err := time.Parse(time.RFC3339Nano, stamp); err == nil {
			stamp = tm.Local().Format("15:04:05")
		}
		event := commandEvent{Kind: "started", ID: shortID(p.CallID), Time: stamp, Description: detail}
		event.Key = key
		event.CallID = p.CallID
		event.ToolName = p.Name
		event.SourceFile = path
		event.StartedAt = r.Timestamp
		event.RawInput = input
		if t.sessionInfo != nil {
			event.Session = t.sessionInfo(path)
		} else {
			event.Session.ID = sessionIDFromPath(path)
		}
		if t.sessionTitle != nil {
			event.SessionTitle = t.sessionTitle(path)
		}
		if slots, ok := t.plans[key]; ok && len(slots) > 0 {
			for _, s := range slots {
				if s != "" {
					event.Commands = append(event.Commands, s)
				}
			}
		} else {
			event.Commands = []string{detail}
		}
		if t.pending == nil {
			t.pending = map[string]commandEvent{}
		}
		t.pending[key] = event
		if t.emit != nil {
			t.emit(event)
		}
	case "function_call_output", "custom_tool_call_output":
		if !t.calls[key] {
			return
		}
		delete(t.calls, key)
		event := t.pending[key]
		delete(t.pending, key)
		event.Kind = "result"
		event.CompletedAt = r.Timestamp
		event.Output = readableOutput(p.Output)
		if slots, ok := t.plans[key]; ok {
			if paired, matched := sequenceResults(slots, p.Output); matched {
				event.Results = paired
				event.Paired = true
			} else {
				event.Output = "无法确认逐条对应关系，以下为整组结果：\n" + event.Output
			}
		} else {
			event.Results = []commandResult{{Description: event.Description, Output: event.Output, Execution: executionFromOutput(p.Output)}}
			event.Paired = true
		}
		delete(t.plans, key)
		if t.emit != nil {
			t.emit(event)
		}
	}
}

func watch(ctx context.Context, root string, history bool, interval time.Duration, emit func(commandEvent)) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer w.Close()
	t := &tracer{files: map[string]*cursor{}, calls: map[string]bool{}, history: history}
	t.emit = emit
	t.sessionTitle = sessionTitleLookup(root)
	t.sessionInfo = sessionInfoLookup()
	dirs := map[string]bool{}
	scan := func(initial bool) error {
		return filepath.WalkDir(root, func(path string, d fs.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if d.IsDir() {
				if !dirs[path] {
					if err := w.Add(path); err != nil {
						return err
					}
					dirs[path] = true
				}
				return nil
			}
			if strings.HasSuffix(path, ".jsonl") {
				if err := t.read(path, initial); err != nil {
					log.Printf("读取失败: %v", err)
				}
			}
			return nil
		})
	}
	if err = scan(true); err != nil {
		return err
	}
	log.Printf("正在监听 %s（Ctrl+C 停止）", root)
	tick := time.NewTicker(interval)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case e, ok := <-w.Events:
			if !ok {
				return nil
			}
			if e.Op&(fsnotify.Remove|fsnotify.Rename) != 0 {
				delete(dirs, e.Name)
			}
			if e.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Rename|fsnotify.Remove) != 0 {
				if err := scan(false); err != nil {
					log.Printf("扫描失败: %v", err)
				}
			}
		case err, ok := <-w.Errors:
			if !ok {
				return nil
			}
			log.Printf("监听错误: %v", err)
		case <-tick.C:
			if err := scan(false); err != nil {
				log.Printf("补查失败: %v", err)
			}
		}
	}
}

func main() {
	home := os.Getenv("CODEX_HOME")
	if home == "" {
		u, err := os.UserHomeDir()
		if err != nil {
			log.Fatal(err)
		}
		home = filepath.Join(u, ".codex")
	}
	root := flag.String("dir", filepath.Join(home, "sessions"), "日志目录")
	history := flag.Bool("history", false, "读取启动前的历史记录")
	interval := flag.Duration("interval", 3*time.Second, "补查间隔")
	cli := flag.Bool("cli", false, "使用控制台模式")
	demo := flag.Bool("demo", false, "在 GUI 中播放演示命令")
	flag.Parse()
	if *interval <= 0 {
		log.Fatal("interval 必须大于 0")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if !*cli {
		runGUI(ctx, *root, *history, *interval, *demo)
		return
	}
	if err := run(ctx, *root, *history, *interval); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, root string, history bool, interval time.Duration) error {
	events := make(chan commandEvent, 256)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for event := range events {
			renderEvent(os.Stdout, event)
		}
	}()
	err := watch(ctx, root, history, interval, func(e commandEvent) { events <- e })
	close(events)
	<-done
	return err
}
