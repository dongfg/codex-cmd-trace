package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"strings"
	"testing"
)

func TestCardOpensReusableFullOutputDetails(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	main := a.NewWindow("main")
	main.SetPadded(false)
	main.Resize(fyne.NewSize(480, 854))
	header := canvas.NewRectangle(ink)
	header.Resize(fyne.NewSize(480, 80))
	viewer := &commandDetails{app: a, main: main, header: header}
	c := newTraceCard("one", "12:00:00", "go test ./...")
	c.onOpen = func() { viewer.Open(c) }
	c.onResult = func() { viewer.Update(c) }
	test.Tap(c.box.Objects[0].(fyne.Tappable))
	if viewer.current != c || viewer.window == nil {
		t.Fatal("card did not open details")
	}
	if got := viewer.window.Canvas().Size(); got != detailWindowSize(main.Canvas().Size(), 80) {
		t.Fatalf("wrong detail size: %v", got)
	}
	if !strings.Contains(viewer.output.String(), "等待") {
		t.Fatal("pending state missing")
	}
	full := "成功\n" + strings.Repeat("line\n", 40) + "LAST LINE"
	c.finish(full)
	if viewer.output.String() != full {
		t.Fatal("detail output was truncated or did not update")
	}
	firstWindow := viewer.window
	other := newTraceCard("two", "", "failed command")
	other.finish("失败（退出码 1）\nFAIL")
	viewer.Open(other)
	if viewer.window != firstWindow || viewer.current != other {
		t.Fatal("details window was not reused")
	}
	if viewer.command.Segments[0].(*widget.TextSegment).Style.ColorName != theme.ColorNameError {
		t.Fatal("failed command color lost")
	}
	viewer.Close()
	if viewer.window != nil || viewer.current != nil {
		t.Fatal("closed detail state retained")
	}
}
