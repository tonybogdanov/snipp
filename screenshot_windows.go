package main

import (
	"errors"
	"image"
	"image/color"
	"syscall"
	"unsafe"
)

var (
	gdi32 = syscall.NewLazyDLL("gdi32.dll")

	procGetDesktopWindow = user32.NewProc("GetDesktopWindow")
	procGetDC            = user32.NewProc("GetDC")
	procReleaseDC        = user32.NewProc("ReleaseDC")
	procGetSystemMetrics = user32.NewProc("GetSystemMetrics")

	procCreateCompatibleDC     = gdi32.NewProc("CreateCompatibleDC")
	procCreateCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	procSelectObject           = gdi32.NewProc("SelectObject")
	procBitBlt                 = gdi32.NewProc("BitBlt")
	procGetDIBits              = gdi32.NewProc("GetDIBits")
	procDeleteDC               = gdi32.NewProc("DeleteDC")
	procDeleteObject           = gdi32.NewProc("DeleteObject")
)

const (
	smXVirtualScreen  = 76
	smYVirtualScreen  = 77
	smCXVirtualScreen = 78
	smCYVirtualScreen = 79
	srcCopy           = 0x00CC0020
	biRGB             = 0
	dibRGBColors      = 0
)

type bitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

type bitmapInfo struct {
	Header bitmapInfoHeader
	Colors [1]uint32
}

func doScreenshot() (string, error) {
	img, err := captureScreen()
	if err != nil {
		return "", err
	}
	return saveScreenshot(img)
}

// captureScreen grabs the full virtual desktop (every monitor, at each
// monitor's true physical resolution — see initDPIAwareness) via GDI
// BitBlt. The virtual screen's origin can be negative (a monitor placed
// left of or above the primary one), so it's read explicitly rather than
// assuming (0,0).
func captureScreen() (image.Image, error) {
	originX, _, _ := procGetSystemMetrics.Call(uintptr(smXVirtualScreen))
	originY, _, _ := procGetSystemMetrics.Call(uintptr(smYVirtualScreen))
	width, _, _ := procGetSystemMetrics.Call(uintptr(smCXVirtualScreen))
	height, _, _ := procGetSystemMetrics.Call(uintptr(smCYVirtualScreen))
	w, h := int(width), int(height)
	if w == 0 || h == 0 {
		return nil, errors.New("could not determine screen size")
	}
	ox, oy := int32(originX), int32(originY)

	desktop, _, _ := procGetDesktopWindow.Call()
	srcDC, _, _ := procGetDC.Call(desktop)
	defer procReleaseDC.Call(desktop, srcDC)

	memDC, _, _ := procCreateCompatibleDC.Call(srcDC)
	defer procDeleteDC.Call(memDC)

	bitmap, _, _ := procCreateCompatibleBitmap.Call(srcDC, uintptr(w), uintptr(h))
	defer procDeleteObject.Call(bitmap)

	oldObj, _, _ := procSelectObject.Call(memDC, bitmap)
	defer procSelectObject.Call(memDC, oldObj)

	ok, _, _ := procBitBlt.Call(memDC, 0, 0, uintptr(w), uintptr(h), srcDC, uintptr(ox), uintptr(oy), uintptr(srcCopy))
	if ok == 0 {
		return nil, errors.New("BitBlt failed")
	}

	var bi bitmapInfo
	bi.Header.Size = uint32(unsafe.Sizeof(bi.Header))
	bi.Header.Width = int32(w)
	bi.Header.Height = int32(-h) // negative = top-down DIB, matches image.RGBA row order
	bi.Header.Planes = 1
	bi.Header.BitCount = 32
	bi.Header.Compression = biRGB

	buf := make([]byte, w*h*4)
	ret, _, _ := procGetDIBits.Call(
		memDC, bitmap, 0, uintptr(h),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&bi)),
		uintptr(dibRGBColors),
	)
	if ret == 0 {
		return nil, errors.New("GetDIBits failed")
	}

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := (y*w + x) * 4
			b, g, r, a := buf[i], buf[i+1], buf[i+2], buf[i+3]
			if a == 0 {
				a = 255
			}
			img.SetRGBA(x, y, color.RGBA{r, g, b, a})
		}
	}
	return img, nil
}
