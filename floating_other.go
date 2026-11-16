//go:build !windows

package main

import (
	"fmt"
	"fyne.io/fyne/v2"
)

func readFloatingDesktop(main fyne.Window) (floatingDesktop, error) {
	return floatingDesktop{}, fmt.Errorf("当前平台尚未适配独立窗口的多屏定位")
}
func placeFloatingWindow(w fyne.Window, rect screenRect, pinned bool) {}
func setFloatingOpacity(w fyne.Window, alpha uint8)                   {}
