package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"math"
)

// Short output uses its natural height. Only long output reaches the viewport cap.
type growingOutputLayout struct {
	content                      fyne.CanvasObject
	maximum, firstLine, progress float32
	startHeight                  float32
}

type tightTextTheme struct{ fyne.Theme }

func (t tightTextTheme) Size(name fyne.ThemeSizeName) float32 {
	if name == theme.SizeNameInnerPadding || name == theme.SizeNamePadding {
		return 0
	}
	return t.Theme.Size(name)
}

type cardTap struct {
	widget.BaseWidget
	content fyne.CanvasObject
	onTap   func()
}

func newCardTap(content fyne.CanvasObject, onTap func()) *cardTap {
	t := &cardTap{content: content, onTap: onTap}
	t.ExtendBaseWidget(t)
	return t
}
func (t *cardTap) CreateRenderer() fyne.WidgetRenderer { return widget.NewSimpleRenderer(t.content) }
func (t *cardTap) Tapped(*fyne.PointEvent) {
	if t.onTap != nil {
		t.onTap()
	}
}
func (t *cardTap) Cursor() desktop.Cursor { return desktop.PointerCursor }

func (l *growingOutputLayout) height() float32 {
	target := float32(math.Min(float64(l.maximum), float64(l.content.MinSize().Height)))
	start := float32(math.Min(float64(l.firstLine), float64(target)))
	if l.startHeight > 0 {
		start = float32(math.Min(float64(l.startHeight), float64(target)))
	}
	return start + (target-start)*l.progress
}
func (l *growingOutputLayout) MinSize([]fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(0, l.height())
}
func (l *growingOutputLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objects {
		o.Move(fyne.NewPos(0, 0))
		o.Resize(fyne.NewSize(size.Width, l.height()))
	}
}

type dragTitle struct {
	widget.BaseWidget
	title  fyne.CanvasObject
	window fyne.Window
	move   func()
}

func newDragTitle(title fyne.CanvasObject, w fyne.Window) *dragTitle {
	d := &dragTitle{title: title, window: w}
	d.ExtendBaseWidget(d)
	return d
}
func (d *dragTitle) CreateRenderer() fyne.WidgetRenderer { return widget.NewSimpleRenderer(d.title) }
func (d *dragTitle) MouseDown(e *desktop.MouseEvent) {
	if e.Button == desktop.MouseButtonPrimary {
		d.move = beginTraceDrag(d.window, e.AbsolutePosition)
	}
}
func (d *dragTitle) MouseUp(*desktop.MouseEvent) { d.DragEnd() }
func (d *dragTitle) Dragged(*fyne.DragEvent) {
	if d.move != nil {
		d.move()
	}
}
func (d *dragTitle) DragEnd() {
	if d.move != nil {
		d.move()
		d.move = nil
	}
}

// The fade overlay is decorative; wheel gestures still belong to the output.
type outputViewport struct {
	widget.BaseWidget
	scroll *container.Scroll
	mask   *canvas.Raster
}

func newOutputViewport(scroll *container.Scroll, mask *canvas.Raster) *outputViewport {
	v := &outputViewport{scroll: scroll, mask: mask}
	v.ExtendBaseWidget(v)
	return v
}
func (v *outputViewport) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewStack(v.scroll, v.mask))
}
func (v *outputViewport) Scrolled(e *fyne.ScrollEvent) { v.scroll.Scrolled(e) }

type verticalCenterLayout struct{ padding float32 }

func (l *verticalCenterLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	min := fyne.NewSize(0, 0)
	for _, o := range objects {
		min = min.Max(o.MinSize())
	}
	min.Height += 2 * l.padding
	return min
}
func (l *verticalCenterLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objects {
		o.Resize(fyne.NewSize(size.Width, o.MinSize().Height))
		height := o.MinSize().Height
		o.Resize(fyne.NewSize(size.Width, height))
		o.Move(fyne.NewPos(0, (size.Height-height)/2))
	}
}
