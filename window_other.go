//go:build !windows

package main

import (
	"fmt"
	"fyne.io/fyne/v2"
)

// The Fyne theme remains available when native transparency isn't supported.
func applyWindowEffect(w fyne.Window) {}

func bringTraceFront(w fyne.Window) { w.RequestFocus() }

func placeDetailsBeside(main, detail fyne.Window, headerHeight float32) { detail.CenterOnScreen() }

func supportsWindowCutout() bool { return false }

func compactWindowSize(w fyne.Window) fyne.Size { return fyne.NewSize(440, 440*16.0/9) }

func setTracePinned(w fyne.Window, pinned bool) error {
	return fmt.Errorf("此平台暂不支持置顶")
}

func beginTraceDrag(w fyne.Window, position fyne.Position) func() { return nil }
