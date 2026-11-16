package main

type screenRect struct{ Left, Top, Right, Bottom int32 }

func clampScreen(value, low, high int32) int32 {
	if high < low {
		return low
	}
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

// Prefer a side that fits; otherwise keep the full detail window visible with minimum overlap.
func detailPlacement(main, work screenRect, width, height, header, gap int32) screenRect {
	if width > work.Right-work.Left {
		width = work.Right - work.Left
	}
	if height > work.Bottom-work.Top {
		height = work.Bottom - work.Top
	}
	left := main.Left - work.Left - gap
	right := work.Right - main.Right - gap
	var x int32
	if left >= width && (right < width || left > right) {
		x = main.Left - gap - width
	} else if right >= width || right >= left {
		x = main.Right + gap
	} else {
		x = main.Left - gap - width
	}
	x = clampScreen(x, work.Left, work.Right-width)
	y := clampScreen(main.Top+header, work.Top, work.Bottom-height)
	return screenRect{x, y, x + width, y + height}
}
