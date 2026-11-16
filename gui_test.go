package main

import (
	"fyne.io/fyne/v2"
	"image/color"
	"strings"
	"testing"
	"unicode/utf8"

	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func TestCommandAnimationDefersOutput(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	c := newTraceCard("test", "12:00:00", "命令: echo 中文🚀")
	c.finish("成功 · 0.1 秒\n中文🚀")
	if c.output.Visible() {
		t.Fatal("output appeared before command animation")
	}
	prev := 0
	for c.visible < len(c.full) {
		c.advance()
		value := c.command.Segments[0].(*widget.TextSegment).Text
		if !utf8.ValidString(value) || c.visible <= prev {
			t.Fatal("invalid character animation", value)
		}
		prev = c.visible
	}
	if !c.output.Visible() || strings.Contains(c.output.String(), "中文🚀") {
		t.Fatal("output was not revealed one line at a time")
	}
	for i := 0; i < outputLineFrames-1; i++ {
		c.advance()
		if strings.Contains(c.output.String(), "中文🚀") {
			t.Fatal("next line appeared too early")
		}
	}
	for c.advance() {
	}
	if !strings.Contains(c.output.String(), "中文🚀") {
		t.Fatal("last line was not revealed")
	}
	if c.status.Text != "● SUCCESS" {
		t.Fatal("wrong status")
	}
}

func TestOutputViewportAndScrollAnimation(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	c := newTraceCard("test", "12:00:00", "echo test", "会话标签")
	c.finish(strings.Repeat("output line\n", 40) + "LAST LINE")
	for c.visible < len(c.full) {
		c.advance()
	}
	height := c.outputBox.MinSize().Height
	c.outputBox.Resize(fyne.NewSize(360, height))
	c.outputScroll.Refresh()
	first := c.outputScroll.Offset.Y
	frames := 0
	sawBothFades := false
	previousShown := c.outputShown
	for c.advance() {
		frames++
		if c.outputShown-previousShown > 1 {
			t.Fatal("multiple lines appeared together")
		}
		previousShown = c.outputShown
		if c.fadeTop && c.fadeBottom {
			sawBothFades = true
		}
		if frames > len(c.outputLines)*(outputLineFrames+1) {
			t.Fatal("animation did not finish")
		}
	}
	if frames < len(c.outputLines)*outputLineFrames || c.outputScroll.Offset.Y <= first || !sawBothFades {
		t.Fatalf("scroll did not animate: %d %v", frames, c.outputScroll.Offset)
	}
	if c.outputBox.MinSize().Height != c.outputLayout.maximum || c.outputScroll.Size().Height != c.outputLayout.maximum {
		t.Fatal("long output did not stop growing at its cap")
	}
	if !strings.HasSuffix(c.output.String(), "LAST LINE") {
		t.Fatal("output lost")
	}
	if !c.fadeTop || c.fadeBottom {
		t.Fatalf("wrong final edge fades: top=%v bottom=%v", c.fadeTop, c.fadeBottom)
	}
}

func TestFailedCommandKeepsErrorColorDuringTyping(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	c := newTraceCard("fail", "12:00:00", "go test ./...")
	c.advance()
	c.finish("失败（退出码 1）\nFAIL")
	for c.visible < len(c.full) {
		c.advance()
		if c.command.Segments[0].(*widget.TextSegment).Style.ColorName != theme.ColorNameError {
			t.Fatal("failure color reset during typing")
		}
	}
	if c.status.Color != failedRed {
		t.Fatal("command and failure badge use different colors")
	}
}

func TestShortOutputUsesNaturalHeight(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	c := newTraceCard("short", "12:00:00", "line one\nline two\nline three")
	initial := c.box.MinSize().Height
	c.finish("成功 · 0.01 秒\nhello")
	for c.visible < len(c.full) {
		c.advance()
	}
	if c.box.MinSize().Height <= initial {
		t.Fatal("card did not grow with command")
	}
	start := c.outputBox.MinSize().Height
	c.outputBox.Resize(fyne.NewSize(360, start))
	for c.advance() {
	}
	final := c.outputBox.MinSize().Height
	if final <= start || final >= c.outputLayout.maximum {
		t.Fatalf("short output reserved blank rows: start=%v final=%v max=%v", start, final, c.outputLayout.maximum)
	}
	if final != c.output.MinSize().Height {
		t.Fatal("short output did not use natural height")
	}
	if c.fadeTop || c.fadeBottom || c.outputScroll.Offset.Y != 0 {
		t.Fatal("short output scrolled or faded unnecessarily")
	}
}

func TestOutputFadesOnlyContextRows(t *testing.T) {
	alpha := func(y int, top, bottom bool) uint8 {
		return color.NRGBAModel.Convert(outputFadeColor(y, 140, 20, top, bottom)).(color.NRGBA).A
	}
	if alpha(0, true, true) == 0 || alpha(139, true, true) == 0 {
		t.Fatal("edges not faded")
	}
	for y := 20; y < 120; y++ {
		if alpha(y, true, true) != 0 {
			t.Fatal("middle five lines were faded")
		}
	}
	if alpha(139, true, false) != 0 || alpha(0, true, false) == 0 {
		t.Fatal("bottom fade did not disappear at end")
	}
}

func TestFooterVerticalCentering(t *testing.T) {
	for _, height := range []float32{18, 36} {
		child := canvas.NewRectangle(color.Transparent)
		child.SetMinSize(fyne.NewSize(100, height))
		l := &verticalCenterLayout{padding: 6}
		size := l.MinSize([]fyne.CanvasObject{child})
		l.Layout([]fyne.CanvasObject{child}, size)
		top := child.Position().Y
		bottom := size.Height - top - child.Size().Height
		if top != bottom || top != 6 {
			t.Fatalf("asymmetric footer: top=%v bottom=%v", top, bottom)
		}
	}
}

func TestWindowCornerCutout(t *testing.T) {
	if !outsideRoundedCorner(0, 0, 480, 854, 10) || !outsideRoundedCorner(479, 853, 480, 854, 10) {
		t.Fatal("square corners remain")
	}
	for _, p := range [][2]int{{10, 0}, {0, 10}, {240, 0}, {240, 400}} {
		if outsideRoundedCorner(p[0], p[1], 480, 854, 10) {
			t.Fatal("interior was clipped", p)
		}
	}
}

func TestPinStatesUseDistinctImageCacheKeys(t *testing.T) {
	off, on := pinIcon(false), pinIcon(true)
	if off.Name() == on.Name() {
		t.Fatal("pin activation can reuse the inactive cached image")
	}
	if string(off.Content()) == string(on.Content()) {
		t.Fatal("pin states have identical artwork")
	}
}
