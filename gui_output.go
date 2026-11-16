package main

import (
	"strings"
	"unicode/utf8"
)

const cardRenderLines = 16
const cardRenderBytes = 8192

func utf8Tail(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	start := len(s) - limit
	for start < len(s) && !utf8.RuneStart(s[start]) {
		start++
	}
	return "…" + s[start:]
}

// The full result remains on traceCard.result. Rendering work does not grow with log age.
func renderedOutputTail(lines []string, shown int) (string, int) {
	if shown > len(lines) {
		shown = len(lines)
	}
	start := shown - cardRenderLines
	if start < 0 {
		start = 0
	}
	parts := make([]string, 0, shown-start)
	for _, line := range lines[start:shown] {
		parts = append(parts, utf8Tail(line, 2048))
	}
	return utf8Tail(strings.Join(parts, "\n"), cardRenderBytes), start
}
