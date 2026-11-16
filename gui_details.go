package main

import (
	"image/color"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type commandDetails struct {
	app                fyne.App
	main               fyne.Window
	header             fyne.CanvasObject
	window             fyne.Window
	current            *traceCard
	command, output    *widget.RichText
	status             *canvas.Text
	metadata, rawInput *widget.RichText
	scroll             *container.Scroll
}

func detailWindowSize(main fyne.Size, headerHeight float32) fyne.Size {
	return fyne.NewSize(main.Width*2, float32(math.Max(1, float64(main.Height-headerHeight))))
}

func (v *commandDetails) Open(c *traceCard) {
	if v.window == nil {
		v.create()
	}
	v.current = c
	v.Update(c)
	v.window.Resize(detailWindowSize(v.main.Canvas().Size(), v.header.Size().Height))
	v.scroll.ScrollToTop()
	v.window.Show()
	applyWindowEffect(v.window)
	placeDetailsBeside(v.main, v.window, v.header.Size().Height)
	bringTraceFront(v.window)
	v.window.RequestFocus()
}

func (v *commandDetails) create() {
	if d, ok := v.app.Driver().(desktop.Driver); ok {
		v.window = d.CreateSplashWindow()
		v.window.SetTitle("Codex Command Trace · Details")
	} else {
		v.window = v.app.NewWindow("Codex Command Trace · Details")
	}
	w := v.window
	w.SetPadded(false)
	w.SetFixedSize(true)
	w.CenterOnScreen()
	title := text("Command Details", 20, ink)
	title.TextStyle = fyne.TextStyle{Monospace: true}
	close := widget.NewButtonWithIcon("", theme.CancelIcon(), w.Close)
	close.Importance = widget.LowImportance
	copy := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
		if v.current != nil {
			w.Clipboard().SetContent(cardInfoText(v.current) + "\n\n" + string(v.current.full) + "\n\n" + v.current.result)
		}
	})
	copy.Importance = widget.LowImportance
	v.status = text("", 11, muted)
	header := container.NewVBox(container.NewBorder(nil, nil, nil, container.NewHBox(copy, close), newDragTitle(title, w)), v.status)
	v.command = widget.NewRichText()
	v.command.Wrapping = fyne.TextWrapBreak
	v.output = widget.NewRichText()
	v.output.Wrapping = fyne.TextWrapBreak
	v.metadata = widget.NewRichText()
	v.metadata.Wrapping = fyne.TextWrapBreak
	v.rawInput = widget.NewRichText()
	v.rawInput.Wrapping = fyne.TextWrapBreak
	raw := widget.NewAccordion(widget.NewAccordionItem("原始调用输入（同组可能包含多条命令）", v.rawInput))
	content := container.NewVBox(text("SESSION / EXECUTION", 12, cyan), v.metadata, widget.NewSeparator(), text("COMMAND", 12, cyan), v.command, widget.NewSeparator(), text("OUTPUT", 12, cyan), v.output, raw)
	v.scroll = container.NewVScroll(content)
	v.scroll.SetMinSize(fyne.NewSize(1, 100))
	w.SetContent(detailWindowFrame(w, container.NewBorder(header, nil, nil, nil, v.scroll)))
	w.SetOnClosed(func() {
		v.window = nil
		v.current = nil
		v.command = nil
		v.output = nil
		v.status = nil
		v.metadata = nil
		v.rawInput = nil
		v.scroll = nil
	})
}

func (v *commandDetails) Update(c *traceCard) {
	if v.window == nil || v.current != c {
		return
	}
	commandColor := theme.ColorNamePrimary
	if c.failed {
		commandColor = theme.ColorNameError
	}
	v.command.Segments = []widget.RichTextSegment{&widget.TextSegment{Text: string(c.full), Style: widget.RichTextStyle{ColorName: commandColor, TextStyle: fyne.TextStyle{Monospace: true}}}}
	v.command.Refresh()
	v.metadata.Segments = []widget.RichTextSegment{&widget.TextSegment{Text: cardInfoText(c), Style: widget.RichTextStyle{ColorName: theme.ColorNamePlaceHolder, SizeName: theme.SizeNameCaptionText, TextStyle: fyne.TextStyle{Monospace: true}}}}
	v.metadata.Refresh()
	input := c.metadata.RawInput
	if input == "" {
		input = "日志未提供"
	}
	v.rawInput.Segments = []widget.RichTextSegment{&widget.TextSegment{Text: input, Style: widget.RichTextStyle{TextStyle: fyne.TextStyle{Monospace: true}}}}
	v.rawInput.Refresh()
	out := c.result
	if !c.finished {
		out = "等待命令输出…"
	}
	v.output.Segments = []widget.RichTextSegment{&widget.TextSegment{Text: out, Style: widget.RichTextStyle{TextStyle: fyne.TextStyle{Monospace: true}}}}
	v.output.Refresh()
	v.status.Text = c.status.Text
	v.status.Color = c.status.Color
	v.status.Refresh()
	v.scroll.Refresh()
}

func (v *commandDetails) Close() {
	if v.window != nil {
		v.window.Close()
	}
}

func detailWindowFrame(w fyne.Window, content fyne.CanvasObject) *fyne.Container {
	bg := canvas.NewRasterWithPixels(func(x, y, width, height int) color.Color {
		if supportsWindowCutout() {
			radius := float64(cardCornerRadius) * float64(width) / math.Max(1, float64(w.Canvas().Size().Width))
			if outsideRoundedCorner(x, y, width, height, radius) {
				return color.NRGBA{A: 255}
			}
		}
		return color.NRGBA{R: 12, G: 23, B: 34, A: 255}
	})
	border := canvas.NewRectangle(color.Transparent)
	border.CornerRadius = cardCornerRadius
	border.StrokeColor = cardBorder
	border.StrokeWidth = 1
	return container.NewStack(bg, border, container.NewPadded(content))
}
