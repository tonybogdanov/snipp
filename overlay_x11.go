//go:build linux

package main

import (
	"errors"
	"image"
	"time"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/randr"
	"github.com/BurntSushi/xgb/xproto"
)

// x11EscapeKeycode is the evdev-derived X11 keycode for Escape. It's the
// same on virtually every modern X11 setup (evdev keycode 1, +8 for X11's
// keycode offset), so it's used directly instead of pulling in XKB just to
// resolve a keysym.
const x11EscapeKeycode = 9

// showFreezeOverlayX11 paints img (tinted 50% white per monitor) across an
// override-redirect window per monitor so the desktop appears frozen,
// grabbing the keyboard and pointer so nothing behind it is reachable. It
// blocks until overlayDuration elapses or the user presses Escape. See
// screenshot_linux.go, which dispatches to this or showFreezeOverlayWayland
// depending on session type.
func showFreezeOverlayX11(img image.Image) {
	conn, err := xgb.NewConn()
	if err != nil {
		return
	}
	defer conn.Close()

	screen := xproto.Setup(conn).Roots[0]
	root := screen.Root

	rects, err := monitorsX11(conn, root)
	if err != nil {
		return
	}

	setup := xproto.Setup(conn)
	maxDataBytes := int(setup.MaximumRequestLength)*4 - 24
	imgBounds := img.Bounds()

	var wins []xproto.Window
	var gcs []xproto.Gcontext
	for _, r := range rects {
		wid, err := conn.NewId()
		if err != nil {
			continue
		}
		win := xproto.Window(wid)
		if err := xproto.CreateWindowChecked(
			conn, screen.RootDepth, win, root,
			int16(r.Min.X), int16(r.Min.Y), uint16(r.Dx()), uint16(r.Dy()), 0,
			xproto.WindowClassInputOutput, screen.RootVisual,
			xproto.CwOverrideRedirect|xproto.CwEventMask,
			[]uint32{1, xproto.EventMaskKeyPress},
		).Check(); err != nil {
			continue
		}

		gcid, err := conn.NewId()
		if err != nil {
			xproto.DestroyWindow(conn, win)
			continue
		}
		gc := xproto.Gcontext(gcid)
		xproto.CreateGC(conn, gc, xproto.Drawable(win), 0, nil)

		xproto.MapWindow(conn, win)

		crop := r.Intersect(imgBounds)
		if !crop.Empty() {
			paintTintedX11(conn, win, gc, screen.RootDepth, tintWhite(img, crop), maxDataBytes)
		}

		wins = append(wins, win)
		gcs = append(gcs, gc)
	}

	if len(wins) == 0 {
		return
	}

	kbGrab, err := xproto.GrabKeyboard(
		conn, true, root, xproto.TimeCurrentTime,
		xproto.GrabModeAsync, xproto.GrabModeAsync,
	).Reply()
	kbGrabbed := err == nil && kbGrab != nil && kbGrab.Status == xproto.GrabStatusSuccess

	ptrGrab, err := xproto.GrabPointer(
		conn, true, root, 0,
		xproto.GrabModeAsync, xproto.GrabModeAsync,
		root, xproto.CursorNone, xproto.TimeCurrentTime,
	).Reply()
	ptrGrabbed := err == nil && ptrGrab != nil && ptrGrab.Status == xproto.GrabStatusSuccess

	events := make(chan xgb.Event, 8)
	done := make(chan struct{})
	go func() {
		for {
			ev, everr := conn.WaitForEvent()
			if ev == nil && everr == nil {
				close(events)
				return
			}
			select {
			case events <- ev:
			case <-done:
				return
			}
		}
	}()

	timeout := time.After(overlayDuration)
loop:
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				break loop
			}
			if kp, ok := ev.(xproto.KeyPressEvent); ok && kp.Detail == x11EscapeKeycode {
				break loop
			}
		case <-timeout:
			break loop
		}
	}
	close(done)

	if kbGrabbed {
		xproto.UngrabKeyboard(conn, xproto.TimeCurrentTime)
	}
	if ptrGrabbed {
		xproto.UngrabPointer(conn, xproto.TimeCurrentTime)
	}
	for i, win := range wins {
		xproto.FreeGC(conn, gcs[i])
		xproto.DestroyWindow(conn, win)
	}
}

// paintTintedX11 uploads a tinted crop into win via PutImage, chunked by
// scanline so a single request never exceeds the server's advertised
// MaximumRequestLength (PutImage isn't guaranteed to support arbitrarily
// large payloads without BigRequests/MIT-SHM, neither of which is wired up
// here).
func paintTintedX11(conn *xgb.Conn, win xproto.Window, gc xproto.Gcontext, depth byte, tinted *image.RGBA, maxDataBytes int) {
	w, h := tinted.Bounds().Dx(), tinted.Bounds().Dy()
	if w == 0 || h == 0 {
		return
	}

	data := make([]byte, w*h*4)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := tinted.RGBAAt(x, y)
			i := (y*w + x) * 4
			data[i+0] = c.B
			data[i+1] = c.G
			data[i+2] = c.R
			data[i+3] = 0
		}
	}

	rowBytes := w * 4
	maxRows := maxDataBytes / rowBytes
	if maxRows < 1 {
		maxRows = 1
	}

	for y0 := 0; y0 < h; y0 += maxRows {
		rows := maxRows
		if y0+rows > h {
			rows = h - y0
		}
		chunk := data[y0*rowBytes : (y0+rows)*rowBytes]
		xproto.PutImage(
			conn, xproto.ImageFormatZPixmap, xproto.Drawable(win), gc,
			uint16(w), uint16(rows), 0, int16(y0), 0, depth, chunk,
		)
	}
}

// monitorsX11 enumerates every connected monitor's absolute screen
// rectangle via RandR's CRTC-based (1.2) API — this xgb version has no
// RandR 1.5 GetMonitors call, so CRTCs with a zero size (disabled outputs)
// are skipped instead.
func monitorsX11(conn *xgb.Conn, root xproto.Window) ([]image.Rectangle, error) {
	if err := randr.Init(conn); err != nil {
		return nil, err
	}
	res, err := randr.GetScreenResourcesCurrent(conn, root).Reply()
	if err != nil {
		return nil, err
	}

	var rects []image.Rectangle
	for _, crtc := range res.Crtcs {
		info, err := randr.GetCrtcInfo(conn, crtc, res.ConfigTimestamp).Reply()
		if err != nil || info == nil {
			continue
		}
		if info.Width == 0 || info.Height == 0 {
			continue
		}
		rects = append(rects, image.Rect(
			int(info.X), int(info.Y),
			int(info.X)+int(info.Width), int(info.Y)+int(info.Height),
		))
	}
	if len(rects) == 0 {
		return nil, errors.New("no monitors found")
	}
	return rects, nil
}
