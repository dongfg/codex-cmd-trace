package main

import "testing"

func intersects(a, b screenRect) bool {
	return a.Left < b.Right && a.Right > b.Left && a.Top < b.Bottom && a.Bottom > b.Top
}
func assertFloatingLayout(t *testing.T, desktop floatingDesktop, placed []floatingPlacement) {
	t.Helper()
	for i, p := range placed {
		onScreen := false
		for _, a := range desktop.Areas {
			if p.Rect.Left >= a.Work.Left && p.Rect.Right <= a.Work.Right && p.Rect.Top >= a.Work.Top && p.Rect.Bottom <= a.Work.Bottom {
				onScreen = true
			}
		}
		if !onScreen || intersects(p.Rect, desktop.Main) {
			t.Fatal("offscreen or covering control window", p)
		}
		for j := 0; j < i; j++ {
			if intersects(p.Rect, placed[j].Rect) {
				t.Fatal("overlapping independent windows", i, j)
			}
		}
	}
}

func TestFloatingColumnsAndMainWindowExclusion(t *testing.T) {
	desktop := floatingDesktop{Main: screenRect{900, 0, 1200, 700}, Areas: []floatingArea{{screenRect{0, 0, 1200, 800}, 1}}}
	requests := make([]floatingRequest, 10)
	for i := range requests {
		requests[i].Height = 150
	}
	placed := packFloating(desktop, requests, 280)
	if len(placed) != 10 {
		t.Fatal(len(placed))
	}
	assertFloatingLayout(t, desktop, placed)
	if placed[0].Rect.Left != placed[1].Rect.Left || placed[1].Rect.Top <= placed[0].Rect.Top || placed[4].Rect.Left <= placed[3].Rect.Left {
		t.Fatal("not top-to-bottom then left-to-right", placed)
	}
}

func TestFloatingMixedDPINegativeMonitorAndMirroring(t *testing.T) {
	desktop := floatingDesktop{Main: screenRect{800, 0, 1000, 650}, Areas: []floatingArea{{screenRect{0, 0, 1000, 700}, 1}, {screenRect{-1600, -200, 0, 1000}, 1.5}}}
	requests := make([]floatingRequest, 14)
	for i := range requests {
		requests[i].Height = 200
	}
	placed := packFloating(desktop, requests, 300)
	if len(placed) != 14 {
		t.Fatal(len(placed))
	}
	assertFloatingLayout(t, desktop, placed)
	seenSecond := false
	for _, p := range placed {
		if p.Rect.Left < 0 {
			seenSecond = true
			if p.Rect.Right-p.Rect.Left != 450 || p.Scale != 1.5 {
				t.Fatal("DPI scale ignored", p)
			}
		}
	}
	if !seenSecond {
		t.Fatal("secondary monitor unused")
	}
	many := make([]floatingRequest, 100)
	for i := range many {
		many[i].Height = 200
	}
	original := len(packFloating(desktop, many, 300))
	desktop.Areas = append(desktop.Areas, desktop.Areas[0])
	if len(packFloating(desktop, many, 300)) != original {
		t.Fatal("mirrored monitor created overlapping slots")
	}
}

func TestTallFloatingCardIsClippedToWorkArea(t *testing.T) {
	desktop := floatingDesktop{Main: screenRect{900, 0, 1200, 700}, Areas: []floatingArea{{screenRect{0, 0, 1200, 800}, 1}}}
	placed := packFloating(desktop, []floatingRequest{{3000}}, 280)
	if len(placed) != 1 {
		t.Fatal(placed)
	}
	assertFloatingLayout(t, desktop, placed)
	if placed[0].Rect.Bottom-placed[0].Rect.Top != 780 {
		t.Fatal("oversized card not bounded", placed)
	}
}
