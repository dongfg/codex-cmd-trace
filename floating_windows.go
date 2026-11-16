//go:build windows

package main

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver"
	"sort"
	"sync"
	"syscall"
	"unsafe"
)

type enumeratedScreen struct {
	handle uintptr
	area   floatingArea
}

var screenEnumerationMu sync.Mutex
var screenEnumeration []enumeratedScreen
var floatingUser32 = syscall.NewLazyDLL("user32.dll")
var floatingDPI = syscall.NewLazyDLL("shcore.dll").NewProc("GetDpiForMonitor")
var enumScreenCallback = syscall.NewCallback(func(handle, dc, rect, data uintptr) uintptr {
	info := struct {
		Size          uint32
		Monitor, Work screenRect
		Flags         uint32
	}{}
	info.Size = uint32(unsafe.Sizeof(info))
	user := floatingUser32
	if ok, _, _ := user.NewProc("GetMonitorInfoW").Call(handle, uintptr(unsafe.Pointer(&info))); ok != 0 {
		scale := float32(1)
		dpi := floatingDPI
		if dpi.Find() == nil {
			var x, y uint32
			if result, _, _ := dpi.Call(handle, 0, uintptr(unsafe.Pointer(&x)), uintptr(unsafe.Pointer(&y))); result == 0 && x > 0 {
				scale = float32(x) / 96
			}
		}
		screenEnumeration = append(screenEnumeration, enumeratedScreen{handle, floatingArea{Work: info.Work, Scale: scale}})
	}
	return 1
})

func readFloatingDesktop(main fyne.Window) (floatingDesktop, error) {
	native, ok := main.(driver.NativeWindow)
	if !ok {
		return floatingDesktop{}, fmt.Errorf("当前窗口不支持独立窗口定位")
	}
	var result floatingDesktop
	var failure error
	native.RunNative(func(context any) {
		c, ok := context.(driver.WindowsWindowContext)
		if !ok {
			failure = fmt.Errorf("无法读取显示器信息")
			return
		}
		user := floatingUser32
		if ok, _, _ := user.NewProc("GetWindowRect").Call(c.HWND, uintptr(unsafe.Pointer(&result.Main))); ok == 0 {
			failure = fmt.Errorf("无法读取主窗口位置")
			return
		}
		nearest, _, _ := user.NewProc("MonitorFromWindow").Call(c.HWND, 2)
		screenEnumerationMu.Lock()
		defer screenEnumerationMu.Unlock()
		screenEnumeration = nil
		if ok, _, _ := user.NewProc("EnumDisplayMonitors").Call(0, 0, enumScreenCallback, 0); ok == 0 {
			failure = fmt.Errorf("无法枚举显示器")
			return
		}
		screens := append([]enumeratedScreen(nil), screenEnumeration...)
		screenEnumeration = nil
		sort.SliceStable(screens, func(i, j int) bool {
			if (screens[i].handle == nearest) != (screens[j].handle == nearest) {
				return screens[i].handle == nearest
			}
			if screens[i].area.Work.Left != screens[j].area.Work.Left {
				return screens[i].area.Work.Left < screens[j].area.Work.Left
			}
			return screens[i].area.Work.Top < screens[j].area.Work.Top
		})
		factor := float32(1)
		if len(screens) > 0 && screens[0].area.Scale > 0 {
			factor = main.Canvas().Scale() / screens[0].area.Scale
		}
		for _, screen := range screens {
			screen.area.Scale *= factor
			result.Areas = append(result.Areas, screen.area)
		}
	})
	if len(result.Areas) == 0 && failure == nil {
		failure = fmt.Errorf("没有可用显示器")
	}
	return result, failure
}

func placeFloatingWindow(w fyne.Window, rect screenRect, pinned bool) {
	native, ok := w.(driver.NativeWindow)
	if !ok {
		return
	}
	native.RunNative(func(context any) {
		c, ok := context.(driver.WindowsWindowContext)
		if !ok {
			return
		}
		user := floatingUser32
		after := ^uintptr(1)
		if pinned {
			after = ^uintptr(0)
		}
		user.NewProc("ShowWindow").Call(c.HWND, 4)
		user.NewProc("SetWindowPos").Call(c.HWND, after, uintptr(rect.Left), uintptr(rect.Top), uintptr(rect.Right-rect.Left), uintptr(rect.Bottom-rect.Top), 0x210)
	})
}

func setFloatingOpacity(w fyne.Window, alpha uint8) {
	native, ok := w.(driver.NativeWindow)
	if !ok {
		return
	}
	native.RunNative(func(context any) {
		c, ok := context.(driver.WindowsWindowContext)
		if ok {
			floatingUser32.NewProc("SetLayeredWindowAttributes").Call(c.HWND, 0, uintptr(alpha), 3)
		}
	})
}
