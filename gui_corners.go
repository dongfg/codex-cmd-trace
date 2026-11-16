package main

// Match the card's radius while using a native transparent colour key at the window corners.
func outsideRoundedCorner(x, y, width, height int, radius float64) bool {
	px, py := float64(x)+0.5, float64(y)+0.5
	cx, cy := px, py
	if px < radius {
		cx = radius
	} else if px > float64(width)-radius {
		cx = float64(width) - radius
	}
	if py < radius {
		cy = radius
	} else if py > float64(height)-radius {
		cy = float64(height) - radius
	}
	dx, dy := px-cx, py-cy
	return dx*dx+dy*dy > radius*radius
}
