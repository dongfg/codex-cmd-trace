package main

import (
	"fyne.io/fyne/v2"
	"image/color"
	"math"
)

// Mask only the extra context row at either edge; the middle five rows stay clear.
func outputFadeColor(y, height, row int, top, bottom bool) color.Color {
	if row < 1 {
		row = 1
	}
	alpha := 0.0
	if top && y < row {
		alpha = 220 * (1 - float64(y)/float64(row))
	}
	if bottom && y >= height-row {
		alpha = math.Max(alpha, 220*float64(y-(height-row)+1)/float64(row))
	}
	return color.NRGBA{R: 14, G: 28, B: 40, A: uint8(math.Max(0, math.Min(220, alpha)))}
}

func (c *traceCard) refreshFade() {
	target := c.output.MinSize().Height - c.outputScroll.Size().Height
	top := target > 0 && c.outputScroll.Offset.Y > 0.5
	bottom := target > 0 && c.outputScroll.Offset.Y < target-0.5
	if top != c.fadeTop || bottom != c.fadeBottom {
		c.fadeTop = top
		c.fadeBottom = bottom
		c.outputFade.Refresh()
	}
}

func pinIcon(active bool) fyne.Resource {
	fill := "none"
	stroke := "#a4b9c5"
	name := "pin-idle.svg"
	if active {
		name = "pin-active.svg"
		fill = "#09202b"
		stroke = "#09202b"
	}
	return fyne.NewStaticResource(name, []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24"><path d="M8 3h8v2l-1 1v5l3 3v2H6v-2l3-3V6L8 5z" fill="`+fill+`" stroke="`+stroke+`" stroke-width="1.7" stroke-linejoin="round"/><path d="M12 16v5" stroke="`+stroke+`" stroke-width="1.7" stroke-linecap="round"/></svg>`))
}

func independentIcon(active bool) fyne.Resource {
	name, color := "independent-off.svg", "#a4b9c5"
	if active {
		name = "independent-on.svg"
		color = "#09202b"
	}
	return fyne.NewStaticResource(name, []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24"><g fill="none" stroke="`+color+`" stroke-width="1.8" stroke-linejoin="round"><rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/></g></svg>`))
}
