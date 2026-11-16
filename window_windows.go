//go:build windows

package main

import (
	"fmt"
	"log"
	"syscall"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver"
)

// Keep all platform-specific effects outside the Fyne UI and event pipeline.
func applyWindowEffect(w fyne.Window) {
	native, ok := w.(driver.NativeWindow)
	if !ok {
		return
	}
	native.RunNative(func(context any) {
		c, ok := context.(driver.WindowsWindowContext)
		if !ok {
			return
		}
		dark := int32(1)
		syscall.NewLazyDLL("dwmapi.dll").NewProc("DwmSetWindowAttribute").Call(c.HWND, 20, uintptr(unsafe.Pointer(&dark)), 4)
		user := syscall.NewLazyDLL("user32.dll")

		get := user.NewProc("GetWindowLongPtrW")
		set := user.NewProc("SetWindowLongPtrW")
		if unsafe.Sizeof(uintptr(0)) == 4 {
			get = user.NewProc("GetWindowLongW")
			set = user.NewProc("SetWindowLongW")
		}
		index := ^uintptr(19) // GWL_EXSTYLE = -20
		style, _, _ := get.Call(c.HWND, index)
		set.Call(c.HWND, index, style|0x80000)
		result, _, err := user.NewProc("SetLayeredWindowAttributes").Call(c.HWND, 0, 243, 3)
		if result == 0 {
			log.Printf("窗口透明效果不可用: %v", err)
		}
	})
}

func bringTraceFront(w fyne.Window) {
	native, ok := w.(driver.NativeWindow)
	if !ok {
		return
	}
	native.RunNative(func(context any) {
		c, ok := context.(driver.WindowsWindowContext)
		if !ok {
			return
		}
		user := syscall.NewLazyDLL("user32.dll")
		user.NewProc("ShowWindow").Call(c.HWND, 1)
		user.NewProc("BringWindowToTop").Call(c.HWND)
		user.NewProc("SetForegroundWindow").Call(c.HWND)
	})
}

func placeDetailsBeside(main, detail fyne.Window, headerHeight float32) {
	parent, ok := main.(driver.NativeWindow)
	if !ok {
		return
	}
	child, ok := detail.(driver.NativeWindow)
	if !ok {
		return
	}
	parent.RunNative(func(context any) {
		m, ok := context.(driver.WindowsWindowContext)
		if !ok {
			return
		}
		user := syscall.NewLazyDLL("user32.dll")
		var rect screenRect
		if ok, _, _ := user.NewProc("GetWindowRect").Call(m.HWND, uintptr(unsafe.Pointer(&rect))); ok == 0 {
			return
		}
		monitor, _, _ := user.NewProc("MonitorFromWindow").Call(m.HWND, 2)
		info := struct {
			Size          uint32
			Monitor, Work screenRect
			Flags         uint32
		}{}
		info.Size = uint32(unsafe.Sizeof(info))
		if ok, _, _ := user.NewProc("GetMonitorInfoW").Call(monitor, uintptr(unsafe.Pointer(&info))); ok == 0 {
			return
		}
		scale := main.Canvas().Scale()
		size := detail.Canvas().Size()
		target := detailPlacement(rect, info.Work, int32(size.Width*scale), int32(size.Height*scale), int32(headerHeight*scale), int32(12*scale))
		child.RunNative(func(context any) {
			d, ok := context.(driver.WindowsWindowContext)
			if !ok {
				return
			}
			user.NewProc("SetWindowPos").Call(d.HWND, 0, uintptr(target.Left), uintptr(target.Top), uintptr(target.Right-target.Left), uintptr(target.Bottom-target.Top), 0x14)
		})
	})
}

func supportsWindowCutout() bool { return true }

func compactWindowSize(w fyne.Window) fyne.Size {
	user := syscall.NewLazyDLL("user32.dll")
	get := user.NewProc("GetSystemMetrics")
	width, _, _ := get.Call(0)
	height, _, _ := get.Call(1)
	scale := w.Canvas().Scale()
	if scale <= 0 {
		scale = 1
	}
	screenWidth := float32(width) / scale
	screenHeight := (float32(height) - 64) / scale
	size := screenWidth / 4
	if size*16/9 > screenHeight {
		size = screenHeight * 9 / 16
	}
	if size <= 0 {
		size = 440
	}
	return fyne.NewSize(size, size*16/9)
}

func setTracePinned(w fyne.Window, pinned bool) error {
	native, ok := w.(driver.NativeWindow)
	if !ok {
		return fmt.Errorf("无法设置置顶")
	}
	var result error
	native.RunNative(func(context any) {
		c, ok := context.(driver.WindowsWindowContext)
		if !ok {
			result = fmt.Errorf("无法设置置顶")
			return
		}
		after := ^uintptr(1)
		if pinned {
			after = ^uintptr(0)
		} // HWND_NOTOPMOST / HWND_TOPMOST
		user := syscall.NewLazyDLL("user32.dll")
		if ok, _, err := user.NewProc("SetWindowPos").Call(c.HWND, after, 0, 0, 0, 0, 0x3); ok == 0 {
			result = fmt.Errorf("置顶失败: %v", err)
			return
		}
		if pinned {
			user.NewProc("SetForegroundWindow").Call(c.HWND)
		}
	})
	return result
}

// Anchor to the original press in screen coordinates, never to Fyne's moving relative deltas.
func beginTraceDrag(w fyne.Window, position fyne.Position) func() {
	native, ok := w.(driver.NativeWindow)
	if !ok {
		return nil
	}
	var move func()
	native.RunNative(func(context any) {
		c, ok := context.(driver.WindowsWindowContext)
		if !ok {
			return
		}
		user := syscall.NewLazyDLL("user32.dll")
		var origin struct{ X, Y int32 }
		var rect struct{ Left, Top, Right, Bottom int32 }
		if ok, _, _ := user.NewProc("ClientToScreen").Call(c.HWND, uintptr(unsafe.Pointer(&origin))); ok == 0 {
			return
		}
		if ok, _, _ := user.NewProc("GetWindowRect").Call(c.HWND, uintptr(unsafe.Pointer(&rect))); ok == 0 {
			return
		}
		scale := w.Canvas().Scale()
		anchorX := origin.X - rect.Left + int32(position.X*scale)
		anchorY := origin.Y - rect.Top + int32(position.Y*scale)
		move = func() {
			var cursor struct{ X, Y int32 }
			if ok, _, _ := user.NewProc("GetCursorPos").Call(uintptr(unsafe.Pointer(&cursor))); ok == 0 {
				return
			}
			user.NewProc("SetWindowPos").Call(c.HWND, 0, uintptr(cursor.X-anchorX), uintptr(cursor.Y-anchorY), 0, 0, 0x15)
		}
	})
	return move
}
