package main

import "testing"

func TestDetailPlacementUsesAvailableSide(t *testing.T) {
	work := screenRect{0, 0, 1920, 1040}
	cases := []struct {
		name       string
		main, work screenRect
		side       string
	}{
		{"right space", screenRect{40, 100, 520, 954}, work, "right"},
		{"left space", screenRect{1400, 100, 1880, 954}, work, "left"},
		{"center fallback", screenRect{720, 100, 1200, 954}, work, ""},
		{"negative monitor", screenRect{-480, 100, 0, 954}, screenRect{-1920, 0, 0, 1040}, "left"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := detailPlacement(c.main, c.work, 960, 773, 81, 12)
			if got.Left < c.work.Left || got.Right > c.work.Right || got.Top < c.work.Top || got.Bottom > c.work.Bottom {
				t.Fatal("off screen", got)
			}
			if c.side == "left" && got.Right > c.main.Left-12 {
				t.Fatal("left placement overlaps", got)
			}
			if c.side == "right" && got.Left < c.main.Right+12 {
				t.Fatal("right placement overlaps", got)
			}
			if got.Right-got.Left != 960 || got.Bottom-got.Top != 773 {
				t.Fatal("requested size changed", got)
			}
		})
	}
}
