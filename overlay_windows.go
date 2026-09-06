package main

import (
	"errors"
	"image"
	"runtime"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procGetModuleHandleW    = kernel32.NewProc("GetModuleHandleW")
	procEnumDisplayMonitors = user32.NewProc("EnumDisplayMonitors")
	procRegisterClassExW    = user32.NewProc("RegisterClassExW")
	procCreateWindowExW     = user32.NewProc("CreateWindowExW")
	procUnregisterClassW    = user32.NewProc("UnregisterClassW")
	procDestroyWindow       = user32.NewProc("DestroyWindow")
	procShowWindow          = user32.NewProc("ShowWindow")
	procUpdateWindow        = user32.NewProc("UpdateWindow")
	procGetMessageW         = user32.NewProc("GetMessageW")
	procTranslateMessage    = user32.NewProc("TranslateMessage")
	procDispatchMessageW    = user32.NewProc("DispatchMessageW")
	procDefWindowProcW      = user32.NewProc("DefWindowProcW")
	procPostQuitMessage     = user32.NewProc("PostQuitMessage")
	procPostMessageW        = user32.NewProc("PostMessageW")
	procBeginPaint          = user32.NewProc("BeginPaint")
	procEndPaint            = user32.NewProc("EndPaint")
	procSetTimer            = user32.NewProc("SetTimer")
	procKillTimer           = user32.NewProc("KillTimer")
	procSetCapture          = user32.NewProc("SetCapture")
	procReleaseCapture      = user32.NewProc("ReleaseCapture")
	procClipCursor          = user32.NewProc("ClipCursor")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procSetWindowsHookExW   = user32.NewProc("SetWindowsHookExW")
	procUnhookWindowsHookEx = user32.NewProc("UnhookWindowsHookEx")
	procCallNextHookEx      = user32.NewProc("CallNextHookEx")

	procStretchDIBits = gdi32.NewProc("StretchDIBits")
)

const (
	wsPopup       = 0x80000000
	wsVisible     = 0x10000000
	wsExTopmost   = 0x00000008
	wsExToolWin   = 0x00000080
	swShowOverlay = 5

	wmDestroyOv = 0x0002
	wmPaintOv   = 0x000F
	wmTimerOv   = 0x0113
	wmAppOv     = 0x8000

	wmOverlayEscape = wmAppOv + 1

	whKeyboardLl = 13
	hcAction     = 0
	wmKeyDown    = 0x0100
	wmSysKeyDown = 0x0104
	vkEscape     = 0x1B

	overlayTimerID = 1
)

type overlayDIB struct {
	header bitmapInfo
	pixels []byte
	w, h   int
}

func buildOverlayDIB(img *image.RGBA) *overlayDIB {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	pixels := make([]byte, w*h*4)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := img.RGBAAt(x, y)
			i := (y*w + x) * 4
			pixels[i+0] = c.B
			pixels[i+1] = c.G
			pixels[i+2] = c.R
		}
	}
	d := &overlayDIB{pixels: pixels, w: w, h: h}
	d.header.Header.Size = uint32(unsafe.Sizeof(d.header.Header))
	d.header.Header.Width = int32(w)
	d.header.Header.Height = int32(-h)
	d.header.Header.Planes = 1
	d.header.Header.BitCount = 32
	d.header.Header.Compression = biRGB
	return d
}

var (
	overlayDIBs     = map[uintptr]*overlayDIB{}
	overlayHooked   int32
	overlayHwndMain uintptr
)

// monitors enumerates every connected monitor's absolute screen rectangle
// via EnumDisplayMonitors.
func monitors() ([]image.Rectangle, error) {
	var rects []image.Rectangle
	cb := syscall.NewCallback(func(hMonitor, hdc, lprc, lParam uintptr) uintptr {
		r := (*winRect)(unsafe.Pointer(lprc))
		rects = append(rects, image.Rect(int(r.Left), int(r.Top), int(r.Right), int(r.Bottom)))
		return 1
	})
	procEnumDisplayMonitors.Call(0, 0, cb, 0)
	if len(rects) == 0 {
		return nil, errors.New("no monitors found")
	}
	return rects, nil
}

type winRect struct{ Left, Top, Right, Bottom int32 }

// showFreezeOverlay paints img (tinted 25% white) across a borderless,
// always-on-top window per monitor so the desktop appears frozen, grabbing
// all keyboard and mouse input so nothing behind it is reachable. It blocks
// until overlayDuration elapses or the user presses Escape.
func showFreezeOverlay(img image.Image) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	screenRects, err := monitors()
	if err != nil {
		return
	}

	originX, _, _ := procGetSystemMetrics.Call(uintptr(smXVirtualScreen))
	originY, _, _ := procGetSystemMetrics.Call(uintptr(smYVirtualScreen))
	ox, oy := int(originX), int(originY)

	hInstance, _, _ := procGetModuleHandleW.Call(0)
	className, _ := syscall.UTF16PtrFromString("SnippOverlayWindow")

	wc := wndClassExOv{
		lpfnWndProc:   syscall.NewCallback(overlayWndProc),
		hInstance:     syscall.Handle(hInstance),
		lpszClassName: className,
	}
	wc.cbSize = uint32(unsafe.Sizeof(wc))
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	defer procUnregisterClassW.Call(uintptr(unsafe.Pointer(className)), hInstance)

	var hwnds []uintptr
	imgBounds := img.Bounds()
	unionRect := screenRects[0]

	for _, sr := range screenRects {
		unionRect = unionRect.Union(sr)

		crop := image.Rect(sr.Min.X-ox, sr.Min.Y-oy, sr.Max.X-ox, sr.Max.Y-oy).Intersect(imgBounds)
		dib := buildOverlayDIB(tintWhite(img, crop))

		hwnd, _, _ := procCreateWindowExW.Call(
			uintptr(wsExTopmost|wsExToolWin),
			uintptr(unsafe.Pointer(className)),
			0,
			uintptr(wsPopup|wsVisible),
			uintptr(int32(sr.Min.X)), uintptr(int32(sr.Min.Y)),
			uintptr(sr.Dx()), uintptr(sr.Dy()),
			0, 0, hInstance, 0,
		)
		if hwnd == 0 {
			continue
		}
		overlayDIBs[hwnd] = dib
		hwnds = append(hwnds, hwnd)
		procShowWindow.Call(hwnd, swShowOverlay)
		procUpdateWindow.Call(hwnd)
	}

	if len(hwnds) == 0 {
		return
	}

	overlayHwndMain = hwnds[0]
	procSetForegroundWindow.Call(overlayHwndMain)
	procSetCapture.Call(overlayHwndMain)
	procClipCursor.Call(uintptr(unsafe.Pointer(&winRect{
		Left: int32(unionRect.Min.X), Top: int32(unionRect.Min.Y),
		Right: int32(unionRect.Max.X), Bottom: int32(unionRect.Max.Y),
	})))

	hHook, _, _ := procSetWindowsHookExW.Call(whKeyboardLl, syscall.NewCallback(lowLevelKeyboardProc), hInstance, 0)
	atomic.StoreInt32(&overlayHooked, 1)

	procSetTimer.Call(overlayHwndMain, overlayTimerID, uintptr(overlayDuration/time.Millisecond), 0)

	var m msgTOv
	for {
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}
		if m.message == wmTimerOv || m.message == wmOverlayEscape {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}

	atomic.StoreInt32(&overlayHooked, 0)
	if hHook != 0 {
		procUnhookWindowsHookEx.Call(hHook)
	}
	procKillTimer.Call(overlayHwndMain, overlayTimerID)
	procClipCursor.Call(0)
	procReleaseCapture.Call()
	for _, hwnd := range hwnds {
		procDestroyWindow.Call(hwnd)
		delete(overlayDIBs, hwnd)
	}
	overlayHwndMain = 0
}

// lowLevelKeyboardProc runs on the overlay's own thread (WH_KEYBOARD_LL
// callbacks are invoked synchronously from that thread's message pump) and
// swallows every key system-wide while the overlay is up — a real screen
// lock rather than a window that merely sits on top — except Escape, which
// it lets close the overlay early instead of waiting out the full timeout.
func lowLevelKeyboardProc(nCode, wParam, lParam uintptr) uintptr {
	if atomic.LoadInt32(&overlayHooked) == 1 && nCode == hcAction {
		kb := (*kbdllhookstruct)(unsafe.Pointer(lParam))
		if kb.vkCode == vkEscape && (wParam == wmKeyDown || wParam == wmSysKeyDown) {
			procPostMessageW.Call(overlayHwndMain, wmOverlayEscape, 0, 0)
		}
		return 1
	}
	ret, _, _ := procCallNextHookEx.Call(0, nCode, wParam, lParam)
	return ret
}

type kbdllhookstruct struct {
	vkCode, scanCode, flags, time uint32
	dwExtraInfo                   uintptr
}

func overlayWndProc(hwnd, msg, wparam, lparam uintptr) uintptr {
	switch uint32(msg) {
	case wmPaintOv:
		var ps [80]byte
		hdc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps[0])))
		if d, ok := overlayDIBs[hwnd]; ok && len(d.pixels) > 0 {
			procStretchDIBits.Call(
				hdc,
				0, 0, uintptr(d.w), uintptr(d.h),
				0, 0, uintptr(d.w), uintptr(d.h),
				uintptr(unsafe.Pointer(&d.pixels[0])),
				uintptr(unsafe.Pointer(&d.header)),
				dibRGBColors, srcCopy,
			)
		}
		procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps[0])))
		return 0
	case wmDestroyOv:
		return 0
	}
	ret, _, _ := procDefWindowProcW.Call(hwnd, msg, wparam, lparam)
	return ret
}

type wndClassExOv struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     syscall.Handle
	hIcon         syscall.Handle
	hCursor       syscall.Handle
	hbrBackground syscall.Handle
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       syscall.Handle
}

type msgTOv struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      struct{ x, y int32 }
}
