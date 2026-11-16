package main

import (
	"context"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var cyan = color.NRGBA{R: 75, G: 235, B: 215, A: 255}
var muted = color.NRGBA{R: 117, G: 150, B: 164, A: 255}
var ink = color.NRGBA{R: 213, G: 229, B: 235, A: 255}
var cardBorder = color.NRGBA{R: 33, G: 65, B: 79, A: 255}
var failedRed = color.NRGBA{R: 255, G: 132, B: 135, A: 255}

const cardCornerRadius float32 = 10
const outputLineFrames = 5 // 75 ms per line with the 15 ms UI ticker.

type traceTheme struct {
	fyne.Theme
	font fyne.Resource
}

func (t traceTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	switch n {
	case theme.ColorNameBackground:
		return color.NRGBA{R: 9, G: 16, B: 25, A: 255}
	case theme.ColorNameForeground:
		return ink
	case theme.ColorNamePrimary, theme.ColorNameFocus:
		return cyan
	case theme.ColorNameButton:
		return color.NRGBA{R: 20, G: 36, B: 48, A: 255}
	case theme.ColorNameInputBackground:
		return color.NRGBA{R: 13, G: 25, B: 36, A: 255}
	case theme.ColorNameDisabled, theme.ColorNamePlaceHolder:
		return muted
	case theme.ColorNameScrollBar:
		return color.Transparent
	case theme.ColorNameError:
		return failedRed
	case theme.ColorNameForegroundOnPrimary:
		return color.NRGBA{R: 9, G: 32, B: 43, A: 255}
	}
	return t.Theme.Color(n, theme.VariantDark)
}
func (t traceTheme) Font(s fyne.TextStyle) fyne.Resource {
	if t.font != nil && !s.Monospace {
		return t.font
	}
	return t.Theme.Font(s)
}
func (t traceTheme) Size(n fyne.ThemeSizeName) float32 {
	if n == theme.SizeNameScrollBar || n == theme.SizeNameScrollBarSmall {
		return 0
	}
	if n == theme.SizeNameText {
		return 14
	}
	if n == theme.SizeNamePadding {
		return 10
	}
	return t.Theme.Size(n)
}
func systemFont() fyne.Resource {
	var paths []string
	switch runtime.GOOS {
	case "windows":
		paths = []string{filepath.Join(os.Getenv("WINDIR"), "Fonts", "Deng.ttf"), filepath.Join(os.Getenv("WINDIR"), "Fonts", "NotoSansSC-VF.ttf")}
	case "darwin":
		paths = []string{"/System/Library/Fonts/Supplemental/Arial Unicode.ttf"}
	default:
		paths = []string{"/usr/share/fonts/truetype/noto/NotoSansSC-Regular.ttf"}
	}
	for _, p := range paths {
		if b, e := os.ReadFile(p); e == nil {
			return fyne.NewStaticResource(filepath.Base(p), b)
		}
	}
	return nil
}
func text(s string, size float32, c color.Color) *canvas.Text {
	x := canvas.NewText(s, c)
	x.TextSize = size
	return x
}

type traceCard struct {
	command           *widget.RichText
	output            *widget.RichText
	status            *canvas.Text
	box               *fyne.Container
	full              []rune
	visible           int
	result            string
	finished          bool
	outputScroll      *container.Scroll
	outputBox         *fyne.Container
	outputFrames      int
	outputFade        *canvas.Raster
	fadeTop           bool
	fadeBottom        bool
	rowHeight         float32
	outputLayout      *growingOutputLayout
	outputLines       []string
	outputShown       int
	outputFromOffset  float32
	failed            bool
	onOpen            func()
	onResult          func()
	metadata          commandEvent
	commandIndex      int
	commandCount      int
	outputRenderStart int
	execution         executionInfo
}

func newTraceCard(id, stamp, command string, session ...string) *traceCard {
	c := &traceCard{full: []rune(command)}
	c.status = text("● RUNNING", 11, cyan)
	var tag fyne.CanvasObject = layout.NewSpacer()
	if len(session) > 0 && session[0] != "" {
		label := widget.NewLabel("# " + session[0])
		label.Truncation = fyne.TextTruncateEllipsis
		tag = label
	}
	meta := container.NewBorder(nil, nil, text(stamp, 10, muted), c.status, tag)
	c.command = widget.NewRichText(&widget.TextSegment{Text: "▌", Style: widget.RichTextStyle{ColorName: theme.ColorNamePrimary, TextStyle: fyne.TextStyle{Monospace: true}}})
	c.command.Wrapping = fyne.TextWrapBreak
	c.output = widget.NewRichText()
	c.output.Wrapping = fyne.TextWrapBreak
	c.output.Hide()
	c.outputScroll = container.NewVScroll(c.output)
	probe := widget.NewRichText(&widget.TextSegment{Text: "M\nM\nM\nM\nM\nM\nM", Style: widget.RichTextStyle{TextStyle: fyne.TextStyle{Monospace: true}}})
	probe.Refresh()
	height := probe.MinSize().Height
	shortProbe := widget.NewRichText(&widget.TextSegment{Text: "M\nM\nM\nM\nM", Style: widget.RichTextStyle{TextStyle: fyne.TextStyle{Monospace: true}}})
	shortProbe.Refresh()
	c.rowHeight = (height - shortProbe.MinSize().Height) / 2
	c.outputFade = canvas.NewRasterWithPixels(func(x, y, width, height int) color.Color {
		logicalHeight := float32(math.Max(1, float64(c.outputScroll.Size().Height)))
		row := int(c.rowHeight * float32(height) / logicalHeight)
		return outputFadeColor(y, height, row, c.fadeTop, c.fadeBottom)
	})
	c.outputScroll.OnScrolled = func(fyne.Position) { c.refreshFade() }
	oneLine := widget.NewRichText(&widget.TextSegment{Text: "M", Style: widget.RichTextStyle{TextStyle: fyne.TextStyle{Monospace: true}}})
	oneLine.Refresh()
	c.outputLayout = &growingOutputLayout{content: c.output, maximum: height, firstLine: oneLine.MinSize().Height}
	c.outputBox = container.New(c.outputLayout, newOutputViewport(c.outputScroll, c.outputFade))
	c.outputBox.Hide()
	body := container.NewVBox(meta, c.command, c.outputBox)
	bg := canvas.NewRectangle(color.NRGBA{R: 14, G: 28, B: 40, A: 238})
	bg.CornerRadius = cardCornerRadius
	bg.StrokeColor = cardBorder
	bg.StrokeWidth = 1
	c.box = container.NewStack(newCardTap(container.NewStack(bg, container.NewPadded(body)), func() {
		if c.onOpen != nil {
			c.onOpen()
		}
	}))
	return c
}
func (c *traceCard) advance() bool {
	if c.visible >= len(c.full) {
		if c.outputFrames > 0 {
			if c.outputFrames > outputLineFrames {
				if c.outputShown >= len(c.outputLines) {
					c.outputFrames = 0
					return false
				}
				c.outputLayout.startHeight = c.outputLayout.height()
				c.outputFromOffset = c.outputScroll.Offset.Y
				previousStart := c.outputRenderStart
				c.outputShown++
				c.updateOutputText()
				c.outputFromOffset = float32(math.Max(0, float64(c.outputFromOffset-float32(c.outputRenderStart-previousStart)*c.rowHeight)))
				c.outputFrames = 1
			}
			t := math.Min(1, float64(c.outputFrames-1)/float64(outputLineFrames-1))
			eased := 1 - math.Pow(1-t, 3)
			c.outputLayout.progress = float32(eased)
			c.outputBox.Refresh()
			c.box.Refresh()
			c.outputScroll.Refresh()
			target := float32(math.Max(0, float64(c.output.MinSize().Height-c.outputScroll.Size().Height)))
			c.outputScroll.ScrollToOffset(fyne.NewPos(0, c.outputFromOffset+(target-c.outputFromOffset)*float32(eased)))
			c.refreshFade()
			c.outputFrames++
			return true
		}
		return false
	}
	// Short commands reveal one rune per frame; long scripts finish in about 1.5 seconds.
	step := int(math.Max(1, float64(len(c.full))/100))
	c.visible += step
	if c.visible > len(c.full) {
		c.visible = len(c.full)
	}
	s := string(c.full[:c.visible])
	if c.visible < len(c.full) {
		s += "▌"
	}
	commandColor := theme.ColorNamePrimary
	if c.failed {
		commandColor = theme.ColorNameError
	}
	c.command.Segments = []widget.RichTextSegment{&widget.TextSegment{Text: s, Style: widget.RichTextStyle{ColorName: commandColor, TextStyle: fyne.TextStyle{Monospace: true}}}}
	c.command.Refresh()
	if c.visible == len(c.full) && c.finished {
		c.showOutput()
	}
	return true
}
func (c *traceCard) showOutput() {
	c.outputLines = strings.Split(strings.ReplaceAll(strings.TrimRight(c.result, "\r\n"), "\r\n", "\n"), "\n")
	c.outputShown = 1
	c.outputFromOffset = 0
	c.updateOutputText()
	c.output.Show()
	c.outputBox.Show()
	c.outputFrames = 1
	c.outputLayout.progress = 0
	c.outputLayout.startHeight = 0
	c.outputBox.Refresh()
	c.box.Refresh()
}
func (c *traceCard) updateOutputText() {
	value, start := renderedOutputTail(c.outputLines, c.outputShown)
	c.outputRenderStart = start
	c.output.Segments = []widget.RichTextSegment{&widget.TextSegment{Text: value, Style: widget.RichTextStyle{TextStyle: fyne.TextStyle{Monospace: true}}}}
	c.output.Refresh()
}
func (c *traceCard) finish(out string) {
	c.result = out
	c.finished = true
	c.status.Text = "● RECEIVED"
	c.status.Color = muted
	if strings.HasPrefix(out, "成功") {
		c.status.Text = "● SUCCESS"
		c.status.Color = cyan
	} else if strings.HasPrefix(out, "失败") {
		c.status.Text = "● FAILED"
		c.status.Color = failedRed
		c.failed = true
		for _, segment := range c.command.Segments {
			if s, ok := segment.(*widget.TextSegment); ok {
				s.Style.ColorName = theme.ColorNameError
			}
		}
		c.command.Refresh()
	} else if strings.HasPrefix(out, "执行中") {
		c.status.Text = "● STREAMING"
		c.status.Color = cyan
	}
	c.status.Refresh()
	if c.onResult != nil {
		c.onResult()
	}
	if c.visible >= len(c.full) {
		c.showOutput()
	}
}

func runGUI(parent context.Context, root string, history bool, interval time.Duration, demo bool) {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	a := app.NewWithID("io.github.dongfg.codex-cmd-trace")
	if absolute, err := filepath.Abs(root); err == nil {
		root = absolute
	}
	a.Settings().SetTheme(traceTheme{Theme: theme.DefaultTheme(), font: systemFont()})
	var w fyne.Window
	if d, ok := a.Driver().(desktop.Driver); ok {
		w = d.CreateSplashWindow()
		w.SetTitle("Codex Command Trace")
	} else {
		w = a.NewWindow("Codex Command Trace")
	}
	w.SetPadded(false)
	w.SetMaster()
	w.SetFixedSize(true)
	w.Resize(compactWindowSize(w))
	w.CenterOnScreen()
	title := text("Codex Command Trace", 20, ink)
	title.TextStyle = fyne.TextStyle{Monospace: true}
	live := text("", 10, failedRed)
	live.Hide()
	pathLabel := widget.NewRichText(&widget.TextSegment{Text: root, Style: widget.RichTextStyle{ColorName: theme.ColorNamePlaceHolder, SizeName: theme.SizeNameCaptionText, TextStyle: fyne.TextStyle{Monospace: true}}})
	pathLabel.Wrapping = fyne.TextWrapBreak
	exit := widget.NewButtonWithIcon("", theme.CancelIcon(), func() { w.Close() })
	exit.Importance = widget.LowImportance
	pinned := false
	var deck *detachedDeck
	var pin *widget.Button
	pin = widget.NewButtonWithIcon("", pinIcon(false), func() {
		if err := setTracePinned(w, !pinned); err != nil {
			live.Text = "●  " + err.Error()
			live.Show()
			live.Refresh()
			return
		}
		pinned = !pinned
		if deck != nil {
			deck.SetPinned(pinned)
		}
		pin.SetIcon(pinIcon(pinned))
		if pinned {
			pin.Importance = widget.HighImportance
		} else {
			pin.Importance = widget.LowImportance
		}
		pin.Refresh()
	})
	pin.Importance = widget.LowImportance
	var independent *widget.Button
	independent = widget.NewButtonWithIcon("", independentIcon(false), func() {
		if deck == nil {
			return
		}
		if err := deck.SetEnabled(!deck.enabled); err != nil {
			live.Text = err.Error()
			live.Show()
			live.Refresh()
			return
		}
		independent.SetIcon(independentIcon(deck.enabled))
		if deck.enabled {
			independent.Importance = widget.HighImportance
		} else {
			independent.Importance = widget.LowImportance
		}
		independent.Refresh()
	})
	independent.Importance = widget.LowImportance
	compactPath := container.NewThemeOverride(pathLabel, tightTextTheme{Theme: a.Settings().Theme()})
	header := container.New(layout.NewCustomPaddedVBoxLayout(2), container.NewBorder(nil, nil, nil, container.NewHBox(independent, pin, exit), newDragTitle(title, w)), compactPath, live)
	feed := container.NewVBox()
	scroll := container.NewVScroll(feed)
	scroll.SetMinSize(fyne.NewSize(1, 100))
	emptyLabel := widget.NewLabel("等待 Codex 执行命令…")
	emptyLabel.Wrapping = fyne.TextWrapWord
	empty := container.NewVBox(text("AWAITING SIGNAL", 16, cyan), emptyLabel)
	feed.Add(container.NewPadded(empty))
	groups := map[string][]*traceCard{}
	var cards []*traceCard
	followFrames := 0
	line := canvas.NewRectangle(color.NRGBA{R: 37, G: 109, B: 115, A: 255})
	line.SetMinSize(fyne.NewSize(1, 1))
	shield := canvas.NewRectangle(color.NRGBA{R: 9, G: 17, B: 27, A: 242})
	shield.CornerRadius = cardCornerRadius
	top := container.NewVBox(container.NewStack(shield, container.NewPadded(header)), line)
	bg := canvas.NewRasterWithPixels(func(x, y, width, height int) color.Color {
		if supportsWindowCutout() {
			radius := float64(cardCornerRadius) * float64(width) / math.Max(1, float64(w.Canvas().Size().Width))
			if outsideRoundedCorner(x, y, width, height, radius) {
				return color.NRGBA{A: 255}
			}
		}
		if x%48 == 0 || y%48 == 0 {
			return color.NRGBA{R: 15, G: 30, B: 42, A: 255}
		}
		return color.NRGBA{R: 9, G: 17, B: 27, A: 255}
	})
	frame := canvas.NewRectangle(color.Transparent)
	frame.CornerRadius = cardCornerRadius
	frame.StrokeColor = cardBorder
	frame.StrokeWidth = 1
	w.SetContent(container.NewStack(bg, frame, container.NewPadded(container.NewBorder(top, nil, nil, nil, scroll))))
	viewer := &commandDetails{app: a, main: w, header: top}
	deck = newDetachedDeck(a, w, feed, feed.Objects[0], func() float32 { return scroll.Size().Width })
	deck.modeHint = emptyLabel
	add := func(e commandEvent, descriptions []string) []*traceCard {
		var group []*traceCard
		for i, d := range descriptions {
			c := newTraceCard(e.ID, e.Time, d, e.SessionTitle)
			c.metadata = e
			c.commandIndex = i + 1
			c.commandCount = len(descriptions)
			c.onOpen = func() { viewer.Open(c) }
			c.onResult = func() { viewer.Update(c) }
			deck.Add(c)
			group = append(group, c)
			cards = append(cards, c)
		}
		feed.Refresh()
		return group
	}
	consume := func(e commandEvent) {
		key := e.Key
		if key == "" {
			key = e.ID
		}
		if e.Kind == "error" {
			live.Text = "●  LISTENER ERROR"
			live.Show()
			live.Color = color.NRGBA{R: 255, G: 132, B: 135, A: 255}
			live.Refresh()
			g := add(e, []string{"监听器错误"})
			g[0].finish(e.Output)
			cards = deck.Trim(groups, cards)
			return
		}
		if e.Kind == "started" {
			ds := e.Commands
			if len(ds) == 0 {
				ds = []string{e.Description}
			}
			groups[key] = add(e, ds)
		} else {
			g, exists := groups[key]
			if !exists {
				return
			} // A retired command's late output must not recreate its card.
			if e.Paired && len(e.Results) > 0 {
				if len(g) != len(e.Results) {
					for _, old := range g {
						if old != nil {
							deck.Remove(old)
						}
					}
					ds := []string{}
					for _, r := range e.Results {
						ds = append(ds, r.Description)
					}
					g = add(e, ds)
				}
				for i, r := range e.Results {
					if g[i] != nil {
						g[i].execution = r.Execution
						g[i].metadata.CompletedAt = e.CompletedAt
						g[i].metadata.Paired = e.Paired
						g[i].finish(r.Output)
					}
				}
			} else {
				if len(g) > 1 {
					for _, c := range g {
						if c != nil {
							deck.Remove(c)
						}
					}
					g = nil
				}
				if len(g) == 0 {
					g = add(e, []string{e.Description})
				}
				g[0].metadata.CompletedAt = e.CompletedAt
				g[0].metadata.Paired = e.Paired
				g[0].finish(e.Output)
			}
			delete(groups, key)
		}
		cards = deck.Trim(groups, cards)
		followFrames = 4
		scroll.ScrollToTop()
	}
	events := make(chan commandEvent, 256)
	send := func(e commandEvent) {
		select {
		case events <- e:
		case <-ctx.Done():
		}
	}
	producerDone := make(chan struct{})
	go func() {
		defer close(producerDone)
		defer close(events)
		if demo {
			demoFeed(ctx, send)
			return
		}
		if err := watch(ctx, root, history, interval, send); err != nil {
			send(commandEvent{Kind: "error", ID: "system", Output: err.Error()})
		}
	}()
	go func() {
		ticker := time.NewTicker(15 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case e, ok := <-events:
				if !ok {
					events = nil
					continue
				}
				fyne.Do(func() {
					if ctx.Err() == nil {
						consume(e)
					}
				})
			case <-ticker.C:
				fyne.Do(func() {
					if ctx.Err() != nil {
						return
					}
					// RichText height settles after rendering; follow on subsequent frames too.
					if followFrames > 0 {
						scroll.Refresh()
						scroll.ScrollToTop()
						followFrames--
					}
					changed := false
					active := cards[:0]
					for _, c := range cards {
						if c.advance() {
							changed = true
						}
						if c.visible < len(c.full) || !c.finished || c.outputFrames > 0 {
							active = append(active, c)
						}
					}
					cards = active
					if deck.Tick(time.Now()) {
						cards = deck.Trim(groups, cards)
						changed = true
					}
					if changed {
						followFrames = 4
						scroll.ScrollToTop()
					}
				})
			}
		}
	}()
	w.SetOnClosed(func() { cancel(); deck.Shutdown(); viewer.Close() })
	w.Show()
	applyWindowEffect(w)
	bringTraceFront(w)
	w.RequestFocus()
	go func() {
		select {
		case <-parent.Done():
			fyne.Do(func() { w.Close() })
		case <-ctx.Done():
		}
	}()
	a.Run()
	cancel()
	<-producerDone
}
