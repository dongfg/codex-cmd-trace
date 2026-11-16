package main

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

type floatingCard struct {
	card              *traceCard
	window            fyne.Window
	detached, retired bool
	serial            int
	rect              screenRect
	size              fyne.Size
	pinned            bool
}

type detachedDeck struct {
	app             fyne.App
	main            fyne.Window
	feed            *fyne.Container
	empty           fyne.CanvasObject
	width           func() float32
	desktop         func() (floatingDesktop, error)
	entries         []*floatingCard // newest first, including cards still in the control window
	enabled, pinned bool
	serial          int
	closing         []*floatingCard
	closingStart    time.Time
	lastLayout      time.Time
	modeHint        *widget.Label
}

func newDetachedDeck(app fyne.App, main fyne.Window, feed *fyne.Container, empty fyne.CanvasObject, width func() float32) *detachedDeck {
	d := &detachedDeck{app: app, main: main, feed: feed, empty: empty, width: width}
	d.desktop = func() (floatingDesktop, error) { return readFloatingDesktop(main) }
	return d
}

func (d *detachedDeck) Add(c *traceCard) {
	d.serial++
	entry := &floatingCard{card: c, detached: d.enabled, serial: d.serial}
	d.entries = append([]*floatingCard{entry}, d.entries...)
	for len(d.entries) > maxHistoryCards {
		d.evict(d.entries[len(d.entries)-1])
	}
	d.syncMain()
	d.lastLayout = time.Time{}
}

func (d *detachedDeck) SetEnabled(enabled bool) error {
	if enabled == d.enabled {
		return nil
	}
	if enabled {
		desktop, err := d.desktop()
		if err != nil {
			return err
		}
		if len(desktop.Areas) == 0 {
			return fmt.Errorf("没有可用的显示区域")
		}
		d.finishClosing()
		log.Printf("独立窗口模式开启，显示器数量：%d", len(desktop.Areas))
		d.enabled = true // Existing entries deliberately remain where they are.
	} else {
		d.enabled = false
		d.closing = nil
		d.closingStart = time.Time{}
		for i := len(d.entries) - 1; i >= 0; i-- {
			if d.entries[i].detached {
				d.closing = append(d.closing, d.entries[i])
			}
		}
	}
	if d.modeHint != nil {
		if enabled {
			d.modeHint.SetText("独立窗口模式已开启，新命令将在独立窗口显示。")
		} else {
			d.modeHint.SetText("等待 Codex 执行命令…")
		}
	}
	d.lastLayout = time.Time{}
	return nil
}

func (d *detachedDeck) SetPinned(pinned bool) { d.pinned = pinned; d.lastLayout = time.Time{} }

func (d *detachedDeck) Trim(groups map[string][]*traceCard, active []*traceCard) []*traceCard {
	alive := map[fyne.CanvasObject]bool{}
	for _, e := range d.entries {
		alive[e.card.box] = true
	}
	return pruneCardReferences(groups, active, alive)
}

func (d *detachedDeck) syncMain() {
	var objects []fyne.CanvasObject
	for _, e := range d.entries {
		if !e.detached {
			objects = append(objects, e.card.box)
		}
	}
	if len(objects) == 0 && d.empty != nil {
		objects = []fyne.CanvasObject{d.empty}
	}
	d.feed.Objects = objects
	d.feed.Refresh()
}

func (d *detachedDeck) closeWindow(e *floatingCard) {
	if e.window != nil {
		w := e.window
		e.window = nil
		w.SetContent(container.NewWithoutLayout())
		w.SetCloseIntercept(nil)
		w.Close()
	}
}

func (d *detachedDeck) evict(e *floatingCard) {
	e.retired = true
	d.closeWindow(e)
	for i, current := range d.entries {
		if current == e {
			copy(d.entries[i:], d.entries[i+1:])
			d.entries[len(d.entries)-1] = nil
			d.entries = d.entries[:len(d.entries)-1]
			break
		}
	}
}

func (d *detachedDeck) Remove(c *traceCard) {
	for _, e := range d.entries {
		if e.card == c {
			d.evict(e)
			break
		}
	}
	d.syncMain()
}

func (d *detachedDeck) finishClosing() {
	for _, e := range d.closing {
		if !e.retired {
			d.closeWindow(e)
			e.detached = false
		}
	}
	d.closing = nil
	d.closingStart = time.Time{}
	d.syncMain()
}

func (d *detachedDeck) Shutdown() {
	d.enabled = false
	for _, e := range d.entries {
		d.closeWindow(e)
	}
	d.closing = nil
}

func (d *detachedDeck) createWindow(e *floatingCard, width, height float32) {
	if driver, ok := d.app.Driver().(desktop.Driver); ok {
		e.window = driver.CreateSplashWindow()
		e.window.SetTitle(fmt.Sprintf("Codex Command Trace · Card %d", e.serial))
	} else {
		e.window = d.app.NewWindow(fmt.Sprintf("Codex Command Trace · Card %d", e.serial))
	}
	w := e.window
	w.SetPadded(false)
	w.SetFixedSize(true)
	w.SetCloseIntercept(func() {})
	wholeCard := container.NewVScroll(e.card.box)
	wholeCard.SetMinSize(fyne.NewSize(1, 1))
	bg := canvas.NewRasterWithPixels(func(x, y, pixelWidth, pixelHeight int) color.Color {
		if supportsWindowCutout() {
			radius := float64(cardCornerRadius) * float64(pixelWidth) / math.Max(1, float64(w.Canvas().Size().Width))
			if outsideRoundedCorner(x, y, pixelWidth, pixelHeight, radius) {
				return color.NRGBA{A: 255}
			}
		}
		return color.NRGBA{R: 14, G: 28, B: 40, A: 255}
	})
	w.SetContent(container.NewStack(bg, wholeCard))
	w.Resize(fyne.NewSize(width, height))
	w.Show()
	applyWindowEffect(w)
}

func (d *detachedDeck) Tick(now time.Time) bool {
	changed := false
	if len(d.closing) > 0 {
		e := d.closing[0]
		if e.retired {
			d.closing = d.closing[1:]
			d.closingStart = time.Time{}
			return true
		}
		if d.closingStart.IsZero() {
			d.closingStart = now
		}
		progress := math.Min(1, float64(now.Sub(d.closingStart))/float64(80*time.Millisecond))
		if e.window != nil {
			setFloatingOpacity(e.window, uint8(243*(1-progress)))
		}
		if progress >= 1 || e.window == nil {
			d.closeWindow(e)
			e.detached = false
			d.closing = d.closing[1:]
			d.closingStart = time.Time{}
			d.syncMain()
			changed = true
		}
	}
	if !d.enabled || now.Sub(d.lastLayout) < 100*time.Millisecond {
		return changed
	}
	d.lastLayout = now
	var floating []*floatingCard
	for i := len(d.entries) - 1; i >= 0; i-- {
		if d.entries[i].detached {
			floating = append(floating, d.entries[i])
		}
	}
	if len(floating) == 0 {
		return changed
	}
	desktop, err := d.desktop()
	if err != nil {
		return changed
	}
	width := d.width()
	if width < 1 {
		width = d.main.Canvas().Size().Width - 20
	}
	var requests []floatingRequest
	for _, e := range floating {
		e.card.box.Resize(fyne.NewSize(width, e.card.box.MinSize().Height))
		requests = append(requests, floatingRequest{Height: e.card.box.MinSize().Height})
	}
	var placements []floatingPlacement
	for len(floating) > 0 {
		placements = packFloating(desktop, requests, width)
		if len(placements) == len(floating) {
			break
		}
		d.evict(floating[0])
		floating = floating[1:]
		requests = requests[1:]
		changed = true
	}
	for i, e := range floating {
		p := placements[i]
		logical := fyne.NewSize(float32(p.Rect.Right-p.Rect.Left)/p.Scale, float32(p.Rect.Bottom-p.Rect.Top)/p.Scale)
		if e.window == nil {
			d.createWindow(e, logical.Width, logical.Height)
		}
		if e.rect != p.Rect || e.size != logical || e.pinned != d.pinned {
			if e.size != logical {
				e.window.Resize(logical)
			}
			placeFloatingWindow(e.window, p.Rect, d.pinned)
			e.rect = p.Rect
			e.size = logical
			e.pinned = d.pinned
		}
	}
	if changed {
		d.syncMain()
	}
	return changed
}
