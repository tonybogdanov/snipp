//go:build linux

package main

import (
	"image"
	"os"
	"time"

	"github.com/rajveermalviya/go-wayland/wayland/client"
	xdg_shell "github.com/rajveermalviya/go-wayland/wayland/stable/xdg-shell"
	ext_session_lock "github.com/rajveermalviya/go-wayland/wayland/staging/ext-session-lock-v1"
	"golang.org/x/image/draw"
	"golang.org/x/sys/unix"
)

// max32 is min/max for int32, which the builtins don't cover.
func max32(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}

// waylandKeyEscape is the raw evdev keycode for Escape, as delivered
// directly (no +8 offset, unlike X11) by wl_keyboard.key events.
const waylandKeyEscape = 1

type wlOutputInfo struct {
	output *client.Output
	x, y   int32
	w, h   int32
	scale  int32

	surface     *client.Surface
	lockSurface *ext_session_lock.ExtSessionLockSurface
	xdgSurface  *xdg_shell.Surface
	toplevel    *xdg_shell.Toplevel
}

// wlSession is everything bound from the registry that the overlay paths
// need, plus the escape signal from the seat's keyboard.
type wlSession struct {
	compositor *client.Compositor
	shm        *client.Shm
	outputs    []*wlOutputInfo
	escapeCh   chan struct{}
	img        image.Image
}

// showFreezeOverlayWayland covers every output with the tinted capture so
// the desktop appears frozen, by whichever of two mechanisms the compositor
// offers:
//
//   - ext-session-lock-v1, the protocol real screen lockers (swaylock,
//     hyprlock) use. Preferred: once locked, the compositor itself stops
//     every other client from receiving input, rather than this process
//     merely asking to be on top.
//   - a fullscreen xdg-shell toplevel per output, when the compositor
//     doesn't offer the lock. Purely visual — it covers the screen but
//     enforces nothing — and it's what KDE/KWin gets, since KWin does not
//     advertise ext_session_lock_manager_v1 to ordinary clients.
//
// GNOME offers neither to a client (mutter implements no session lock and
// no layer shell), so there the overlay is skipped; the screenshot itself
// still saves. Blocks until overlayDuration elapses or Escape is pressed.
func showFreezeOverlayWayland(img image.Image) {
	display, err := client.Connect("")
	if err != nil {
		debugf("wayland: connect failed: %v", err)
		return
	}
	defer display.Context().Close()

	registry, err := display.GetRegistry()
	if err != nil {
		debugf("wayland: registry failed: %v", err)
		return
	}

	var (
		compositor  *client.Compositor
		shm         *client.Shm
		seat        *client.Seat
		lockManager *ext_session_lock.ExtSessionLockManager
		wmBase      *xdg_shell.WmBase
		outputs     []*wlOutputInfo
	)

	registry.SetGlobalHandler(func(e client.RegistryGlobalEvent) {
		debugf("wayland: global %s v%d", e.Interface, e.Version)

		switch e.Interface {
		case "wl_compositor":
			compositor = client.NewCompositor(display.Context())
			registry.Bind(e.Name, e.Interface, e.Version, compositor)
		case "wl_shm":
			shm = client.NewShm(display.Context())
			registry.Bind(e.Name, e.Interface, e.Version, shm)
		case "wl_seat":
			seat = client.NewSeat(display.Context())
			registry.Bind(e.Name, e.Interface, e.Version, seat)
		case "wl_output":
			output := client.NewOutput(display.Context())
			registry.Bind(e.Name, e.Interface, e.Version, output)
			info := &wlOutputInfo{output: output, scale: 1}
			output.SetGeometryHandler(func(ge client.OutputGeometryEvent) {
				info.x, info.y = ge.X, ge.Y
			})
			output.SetModeHandler(func(me client.OutputModeEvent) {
				info.w, info.h = me.Width, me.Height
			})
			output.SetScaleHandler(func(se client.OutputScaleEvent) {
				if se.Factor > 0 {
					info.scale = se.Factor
				}
			})
			outputs = append(outputs, info)
		case "ext_session_lock_manager_v1":
			lockManager = ext_session_lock.NewExtSessionLockManager(display.Context())
			registry.Bind(e.Name, e.Interface, e.Version, lockManager)
		case "xdg_wm_base":
			wmBase = xdg_shell.NewWmBase(display.Context())
			registry.Bind(e.Name, e.Interface, e.Version, wmBase)
			// A compositor that pings and gets no pong assumes the client
			// has hung and may kill it.
			wmBase.SetPingHandler(func(pe xdg_shell.WmBasePingEvent) {
				wmBase.Pong(pe.Serial)
			})
		}
	})

	wlRoundTrip(display)
	wlRoundTrip(display)

	if compositor == nil || shm == nil || len(outputs) == 0 {
		debugf("wayland: missing basics: compositor=%t shm=%t outputs=%d",
			compositor != nil, shm != nil, len(outputs))
		return
	}
	debugf("wayland: %d output(s), lock_manager=%t xdg_wm_base=%t",
		len(outputs), lockManager != nil, wmBase != nil)

	session := &wlSession{
		compositor: compositor,
		shm:        shm,
		outputs:    outputs,
		escapeCh:   make(chan struct{}, 1),
		img:        img,
	}

	if seat != nil {
		if keyboard, err := seat.GetKeyboard(); err == nil && keyboard != nil {
			keyboard.SetKeyHandler(func(e client.KeyboardKeyEvent) {
				if e.Key == waylandKeyEscape && client.KeyboardKeyState(e.State) == client.KeyboardKeyStatePressed {
					select {
					case session.escapeCh <- struct{}{}:
					default:
					}
				}
			})
		}
	}

	stop := make(chan struct{})
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
			}
			if display.Context().Dispatch() != nil {
				return
			}
		}
	}()
	defer close(stop)

	switch {
	case lockManager != nil:
		session.lockOverlay(lockManager)
	case wmBase != nil:
		session.fullscreenOverlay(wmBase)
	default:
		debugf("wayland: compositor offers neither a session lock nor xdg-shell; skipping the overlay")
	}
}

// lockOverlay covers every output with an ext-session-lock-v1 lock surface.
func (s *wlSession) lockOverlay(manager *ext_session_lock.ExtSessionLockManager) {
	lock, err := manager.Lock()
	if err != nil || lock == nil {
		debugf("wayland: lock request failed: %v", err)
		return
	}

	lockedCh := make(chan struct{}, 1)
	finishedCh := make(chan struct{}, 1)
	lock.SetLockedHandler(func(ext_session_lock.ExtSessionLockLockedEvent) {
		signal(lockedCh)
	})
	lock.SetFinishedHandler(func(ext_session_lock.ExtSessionLockFinishedEvent) {
		signal(finishedCh)
	})

	// The lock surfaces have to be created and painted first: the
	// compositor sends `locked` only once every output is covered by a lock
	// surface with a committed buffer. Waiting for `locked` before creating
	// them deadlocks — the compositor waiting on the client and the client
	// on the compositor.
	for _, o := range s.outputs {
		surface, err := s.compositor.CreateSurface()
		if err != nil {
			debugf("wayland: create surface failed: %v", err)
			continue
		}
		o.surface = surface

		lockSurface, err := lock.GetLockSurface(surface, o.output)
		if err != nil {
			debugf("wayland: get lock surface failed: %v", err)
			continue
		}
		o.lockSurface = lockSurface

		info := o
		lockSurface.SetConfigureHandler(func(ce ext_session_lock.ExtSessionLockSurfaceConfigureEvent) {
			lockSurface.AckConfigure(ce.Serial)
			debugf("wayland: lock surface configure %dx%d", ce.Width, ce.Height)
			s.paint(info, int32(ce.Width), int32(ce.Height))
		})
	}

	locked, aborted := false, false
	select {
	case <-lockedCh:
		debugf("wayland: compositor confirmed the lock")
		locked = true
	case <-finishedCh:
		debugf("wayland: compositor finished the lock (refused or dropped)")
		aborted = true
	case <-time.After(2 * time.Second):
		debugf("wayland: lock never confirmed, keeping the surfaces up anyway")
	}

	if !aborted {
		select {
		case <-s.escapeCh:
		case <-finishedCh:
			locked = false
		case <-time.After(overlayDuration):
		}
	}

	for _, o := range s.outputs {
		if o.lockSurface != nil {
			o.lockSurface.Destroy()
		}
		if o.surface != nil {
			o.surface.Destroy()
		}
	}

	// UnlockAndDestroy is only valid on a lock the compositor confirmed;
	// destroying an unlocked one is the protocol's other exit.
	if locked {
		lock.UnlockAndDestroy()
	} else {
		lock.Destroy()
	}
}

// fullscreenOverlay covers every output with a fullscreen xdg-shell
// toplevel. Used where there's no session lock to be had — it can't stop
// input reaching what's underneath, but it does cover the screen, which is
// what the overlay is for.
func (s *wlSession) fullscreenOverlay(wmBase *xdg_shell.WmBase) {
	for _, o := range s.outputs {
		surface, err := s.compositor.CreateSurface()
		if err != nil {
			debugf("wayland: create surface failed: %v", err)
			continue
		}
		o.surface = surface

		xdgSurface, err := wmBase.GetXdgSurface(surface)
		if err != nil {
			debugf("wayland: get xdg surface failed: %v", err)
			continue
		}
		o.xdgSurface = xdgSurface

		toplevel, err := xdgSurface.GetToplevel()
		if err != nil {
			debugf("wayland: get toplevel failed: %v", err)
			continue
		}
		o.toplevel = toplevel

		toplevel.SetTitle("Snipp")
		toplevel.SetAppId("snipp")
		toplevel.SetFullscreen(o.output)

		// The toplevel's configure carries the size the compositor wants;
		// it arrives before the xdg_surface configure that commits the
		// state, so by the time the paint below runs the size is known.
		// Until then, fall back to the output's own mode — which is in
		// physical pixels, so it has to come back down to logical ones
		// before paint scales it up again.
		info := o
		w, h := o.w/max32(o.scale, 1), o.h/max32(o.scale, 1)
		toplevel.SetConfigureHandler(func(ce xdg_shell.ToplevelConfigureEvent) {
			debugf("wayland: toplevel configure %dx%d", ce.Width, ce.Height)
			if ce.Width > 0 && ce.Height > 0 {
				w, h = ce.Width, ce.Height
			}
		})
		xdgSurface.SetConfigureHandler(func(ce xdg_shell.SurfaceConfigureEvent) {
			xdgSurface.AckConfigure(ce.Serial)
			s.paint(info, w, h)
		})

		// An initial commit with no buffer attached is what asks the
		// compositor for that first configure.
		surface.Commit()
	}

	select {
	case <-s.escapeCh:
	case <-time.After(overlayDuration):
	}

	for _, o := range s.outputs {
		if o.toplevel != nil {
			o.toplevel.Destroy()
		}
		if o.xdgSurface != nil {
			o.xdgSurface.Destroy()
		}
		if o.surface != nil {
			o.surface.Destroy()
		}
	}
}

// paint fills one output's surface with its slice of the capture. w and h
// are the surface size in logical pixels; the buffer is built at the
// output's scale, since the capture is in physical ones.
func (s *wlSession) paint(o *wlOutputInfo, w, h int32) {
	if w <= 0 || h <= 0 || o.surface == nil {
		return
	}

	scale := o.scale
	if scale < 1 {
		scale = 1
	}
	bufW, bufH := w*scale, h*scale

	// The buffer size and the capture's pixel dimensions are arrived at
	// independently — the first from the compositor's configure times the
	// output's integer scale, the second from whatever the portal handed
	// back — and they don't have to agree. Under fractional scaling they
	// reliably don't: KWin advertises scale 1 on wl_output (the integer
	// event can't express 1.5) while the capture is at full device
	// resolution. So take the output's region of the capture on its own
	// terms and resample it to the buffer, rather than cutting out bufW x
	// bufH pixels and trusting that to be the right region.
	tinted := scaleTo(tintWhite(s.img, s.captureRect(o)), bufW, bufH)

	buf := buildWaylandBuffer(s.shm, tinted, bufW, bufH)
	if buf == nil {
		debugf("wayland: buffer allocation failed for %dx%d", bufW, bufH)
		return
	}

	s.surfaceCommit(o.surface, buf, scale, bufW, bufH)
}

// captureRect is the region of the capture that belongs to o, in capture
// pixels. With one output that's the whole capture, which sidesteps the
// guesswork entirely — and one output is the common case. With several,
// wl_output gives the mode in physical pixels but the position only in
// logical ones, so the position is scaled up and the result clamped; an
// empty intersection means the guess was wrong, and the whole capture is a
// better wrong answer than nothing.
func (s *wlSession) captureRect(o *wlOutputInfo) image.Rectangle {
	bounds := s.img.Bounds()
	if len(s.outputs) < 2 || o.w <= 0 || o.h <= 0 {
		return bounds
	}

	scale := o.scale
	if scale < 1 {
		scale = 1
	}

	x, y := int(o.x*scale), int(o.y*scale)
	rect := image.Rect(x, y, x+int(o.w), y+int(o.h)).Intersect(bounds)
	if rect.Empty() {
		debugf("wayland: output region %v falls outside the capture %v", rect, bounds)
		return bounds
	}
	return rect
}

// scaleTo resamples img to exactly w x h, returning it untouched when it's
// already that size (the case whenever the compositor's idea of the output
// matches the capture). ApproxBiLinear rather than a better filter because
// this runs on the dispatch path at up to 4K, and the result is a
// half-white-tinted backdrop where sharpness doesn't show.
func scaleTo(img *image.RGBA, w, h int32) *image.RGBA {
	if img.Bounds().Dx() == int(w) && img.Bounds().Dy() == int(h) {
		return img
	}

	debugf("wayland: resampling the capture from %v to %dx%d", img.Bounds(), w, h)
	out := image.NewRGBA(image.Rect(0, 0, int(w), int(h)))
	draw.ApproxBiLinear.Scale(out, out.Bounds(), img, img.Bounds(), draw.Src, nil)
	return out
}

// surfaceCommit attaches buf and presents it. Attaching alone shows
// nothing: a commit only presents the regions the client marked damaged.
func (s *wlSession) surfaceCommit(surface *client.Surface, buf *client.Buffer, scale, bufW, bufH int32) {
	surface.Attach(buf, 0, 0)
	surface.SetBufferScale(scale)
	surface.DamageBuffer(0, 0, bufW, bufH)
	surface.Commit()
}

// signal delivers on ch without ever blocking, for handlers that run on the
// dispatch goroutine and must not stall it.
func signal(ch chan struct{}) {
	select {
	case ch <- struct{}{}:
	default:
	}
}

// buildWaylandBuffer uploads tinted into a fresh wl_shm-backed buffer sized
// w x h. The backing store is an unlinked temp file (shared memory via a
// file descriptor is the only buffer mechanism wl_shm offers).
func buildWaylandBuffer(shm *client.Shm, tinted *image.RGBA, w, h int32) *client.Buffer {
	stride := w * 4
	size := stride * h
	if size <= 0 {
		return nil
	}

	f, err := os.CreateTemp("", "snipp-wayland-*")
	if err != nil {
		return nil
	}
	os.Remove(f.Name())
	defer f.Close()

	if err := f.Truncate(int64(size)); err != nil {
		return nil
	}

	data, err := unix.Mmap(int(f.Fd()), 0, int(size), unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED)
	if err != nil {
		return nil
	}
	defer unix.Munmap(data)

	tw, th := tinted.Bounds().Dx(), tinted.Bounds().Dy()
	for y := 0; y < int(h); y++ {
		for x := 0; x < int(w); x++ {
			i := int(stride)*y + x*4
			// The capture doesn't necessarily cover the whole output; any
			// strip it doesn't reach reads as part of the white tint rather
			// than as a black band.
			var c = struct{ r, g, b, a byte }{255, 255, 255, 255}
			if x < tw && y < th {
				p := tinted.RGBAAt(x, y)
				c.r, c.g, c.b, c.a = p.R, p.G, p.B, p.A
			}
			// ARGB8888 little-endian in memory as B,G,R,A bytes.
			data[i+0] = c.b
			data[i+1] = c.g
			data[i+2] = c.r
			data[i+3] = c.a
		}
	}

	pool, err := shm.CreatePool(int(f.Fd()), size)
	if err != nil {
		return nil
	}
	defer pool.Destroy()

	buf, err := pool.CreateBuffer(0, w, h, stride, uint32(client.ShmFormatArgb8888))
	if err != nil {
		return nil
	}
	buf.SetReleaseHandler(func(client.BufferReleaseEvent) {
		buf.Destroy()
	})
	return buf
}

// wlRoundTrip blocks until the compositor has processed every request sent
// so far, so subsequently-read state (registry globals, output geometry)
// reflects handlers that have actually run.
func wlRoundTrip(display *client.Display) {
	callback, err := display.Sync()
	if err != nil {
		return
	}
	defer callback.Destroy()

	done := false
	callback.SetDoneHandler(func(client.CallbackDoneEvent) {
		done = true
	})
	for !done {
		if display.Context().Dispatch() != nil {
			return
		}
	}
}
