package main

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestRenderedOutputStaysBounded(t *testing.T) {
	line := strings.Repeat("中文🚀", 3000)
	lines := make([]string, 100000)
	for i := range lines {
		lines[i] = line
	}
	value, start := renderedOutputTail(lines, len(lines))
	if start != len(lines)-cardRenderLines || len(value) > cardRenderBytes+3 || !utf8.ValidString(value) {
		t.Fatalf("unbounded or invalid rendering: %d %d", start, len(value))
	}
	if lines[0] != line {
		t.Fatal("full output was modified")
	}
}

func BenchmarkOutputTail(b *testing.B) {
	for _, count := range []int{100, 10000, 100000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			lines := make([]string, count)
			for i := range lines {
				lines[i] = "one output line with ordinary command output"
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				renderedOutputTail(lines, count)
			}
		})
	}
}
