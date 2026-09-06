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

	procCreateDCW              = gdi32.NewProc("CreateDCW")
	procCreateCompatibleDC     = gdi32.NewProc("CreateCompatibleDC")
	procCreateCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	procSelectObject           = gdi32.NewProc("SelectObject")
	procBitBlt                 = gdi32.NewProc("BitBlt")
	procGetDIBits              = gdi32.NewProc("GetDIBits")
	procDeleteDC               = gdi32.NewProc("DeleteDC")
	procDeleteObject           = gdi32.NewProc("DeleteObject")
)

const (
	srcCopy      = 0x00CC0020
	biRGB        = 0
	dibRGBColors = 0
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

// captureScreen grabs every monitor at its own true physical resolution and
// composites them into one image, sized and laid out to match monitors()
// exactly (both derive from the same EnumDisplayMonitors enumeration).
//
// Each monitor is captured from its own device context (CreateDCW on that
// monitor's device name) rather than one BitBlt over the whole "virtual
// screen" via GetDesktopWindow's DC. The latter is anchored to the primary
// monitor's DPI: on a mixed/scaled-DPI setup, GDI silently rescales
// non-primary monitors to fit that shared coordinate grid, so a captured
// slice for such a monitor doesn't match that monitor's real pixel
// dimensions from EnumDisplayMonitors — which is exactly what showed up as
// a stretched/shrunk freeze overlay on scaled displays. Per-monitor device
// contexts use that monitor's own native pixel grid, sidestepping the
// rescale entirely.
func captureScreen() (image.Image, error) {
	mons, err := enumMonitors()
	if err != nil {
		return nil, err
	}

	union := mons[0].rect
	for _, m := range mons[1:] {
		union = union.Union(m.rect)
	}
	if union.Dx() == 0 || union.Dy() == 0 {
		return nil, errors.New("could not determine screen size")
	}

	img := image.NewRGBA(image.Rect(0, 0, union.Dx(), union.Dy()))

	for _, m := range mons {
		if err := captureMonitorInto(img, m, union.Min); err != nil {
			return nil, err
		}
	}
	return img, nil
}

// captureMonitorInto captures m's own device context and draws it into dst
// at m's position relative to origin.
func captureMonitorInto(dst *image.RGBA, m winMonitor, origin image.Point) error {
	devicePtr, err := syscall.UTF16PtrFromString(m.device)
	if err != nil {
		return err
	}

	monDC, _, _ := procCreateDCW.Call(0, uintptr(unsafe.Pointer(devicePtr)), 0, 0)
	if monDC == 0 {
		return errors.New("CreateDC failed for monitor " + m.device)
	}
	defer procDeleteDC.Call(monDC)

	w, h := m.rect.Dx(), m.rect.Dy()
	if w == 0 || h == 0 {
		return nil
	}

	memDC, _, _ := procCreateCompatibleDC.Call(monDC)
	defer procDeleteDC.Call(memDC)

	bitmap, _, _ := procCreateCompatibleBitmap.Call(monDC, uintptr(w), uintptr(h))
	defer procDeleteObject.Call(bitmap)

	oldObj, _, _ := procSelectObject.Call(memDC, bitmap)
	defer procSelectObject.Call(memDC, oldObj)

	ok, _, _ := procBitBlt.Call(memDC, 0, 0, uintptr(w), uintptr(h), monDC, 0, 0, uintptr(srcCopy))
	if ok == 0 {
		return errors.New("BitBlt failed for monitor " + m.device)
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
		return errors.New("GetDIBits failed for monitor " + m.device)
	}

	dx, dy := m.rect.Min.X-origin.X, m.rect.Min.Y-origin.Y
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := (y*w + x) * 4
			b, g, r, a := buf[i], buf[i+1], buf[i+2], buf[i+3]
			if a == 0 {
				a = 255
			}
			dst.SetRGBA(dx+x, dy+y, color.RGBA{r, g, b, a})
		}
	}
	return nil
}
