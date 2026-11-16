package main

import "math"

type floatingArea struct {
	Work  screenRect
	Scale float32
}
type floatingDesktop struct {
	Main  screenRect
	Areas []floatingArea
}
type floatingRequest struct{ Height float32 }
type floatingPlacement struct {
	Rect  screenRect
	Scale float32
}

// Pack chronologically down each column, then across, then onto the next monitor.
// Columns intersecting the control window use only the free vertical intervals.
func packFloating(desktop floatingDesktop, requests []floatingRequest, width float32) []floatingPlacement {
	var placed []floatingPlacement
	seen := map[screenRect]bool{}
	for _, area := range desktop.Areas {
		if seen[area.Work] {
			continue
		}
		seen[area.Work] = true
		scale := area.Scale
		if scale <= 0 {
			scale = 1
		}
		gap := int32(math.Ceil(float64(10 * scale)))
		tileWidth := int32(math.Ceil(float64(width * scale)))
		work := area.Work
		if tileWidth > work.Right-work.Left-2*gap {
			continue
		}
		for x := work.Left + gap; x+tileWidth <= work.Right-gap; x += tileWidth + gap {
			intervals := [][2]int32{{work.Top + gap, work.Bottom - gap}}
			main := desktop.Main
			if x < main.Right+gap && x+tileWidth > main.Left-gap {
				intervals = nil
				if main.Top-gap > work.Top+gap {
					intervals = append(intervals, [2]int32{work.Top + gap, clampScreen(main.Top-gap, work.Top+gap, work.Bottom-gap)})
				}
				if main.Bottom+gap < work.Bottom-gap {
					intervals = append(intervals, [2]int32{clampScreen(main.Bottom+gap, work.Top+gap, work.Bottom-gap), work.Bottom - gap})
				}
			}
			for _, interval := range intervals {
				y := interval[0]
				for len(placed) < len(requests) {
					height := int32(math.Ceil(float64(requests[len(placed)].Height * scale)))
					maxHeight := work.Bottom - work.Top - 2*gap
					if height > maxHeight {
						height = maxHeight
					}
					if height < 1 {
						height = 1
					}
					if y+height > interval[1] {
						break
					}
					placed = append(placed, floatingPlacement{Rect: screenRect{x, y, x + tileWidth, y + height}, Scale: scale})
					y += height + gap
				}
				if len(placed) == len(requests) {
					return placed
				}
			}
		}
	}
	return placed
}
