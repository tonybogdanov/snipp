//go:build linux

package main

import (
	"image"
	"os"
	"time"

	"github.com/rajveermalviya/go-wayland/wayland/client"
	ext_session_lock "github.com/rajveermalviya/go-wayland/wayland/staging/ext-session-lock-v1"
	"golang.org/x/sys/unix"
)

// waylandKeyEscape is the raw evdev keycode for Escape, as delivered
// directly (no +8 offset, unlike X11) by wl_keyboard.key events.
const waylandKeyEscape = 1

type wlOutputInfo struct {
	output      *client.Output
	x, y        int32
	w, h        int32
	surface     *client.Surface
	lockSurface *ext_session_lock.ExtSessionLockSurface
}

// showFreezeOverlayWayland locks the session via ext-session-lock-v1 —
// the same protocol real screen lockers (swaylock, hyprlock) use — and
// paints a tinted lock surface per output. This gives the strongest "true
// lock" guarantee of any platform here: once locked, the compositor itself
// blocks all other clients from receiving input, rather than this process
// merely asking nicely to be on top. Blocks until overlayDuration elapses
// or Escape is pressed.
//
// Known limitation: wl_output's geometry event gives each output's
// position, but compositors are explicitly allowed to fake or omit it
// (protocol note: "Some compositors ... might fake this information").
// There's no live Wayland environment to verify this against, so this
// assumes the portal's composited screenshot lays out outputs at their
// wl_output logical positions — true for the common compositors (GNOME,
// KDE, wlroots) but not guaranteed by the protocol itself.
func showFreezeOverlayWayland(img image.Image) {
	display, err := client.Connect("")
	if err != nil {
		return
	}
	defer display.Context().Close()

	registry, err := display.GetRegistry()
	if err != nil {
		return
	}

	var (
		compositor  *client.Compositor
		shm         *client.Shm
		seat        *client.Seat
		lockManager *ext_session_lock.ExtSessionLockManager
		outputs     []*wlOutputInfo
	)

	registry.SetGlobalHandler(func(e client.RegistryGlobalEvent) {
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
			info := &wlOutputInfo{output: output}
			output.SetGeometryHandler(func(ge client.OutputGeometryEvent) {
				info.x, info.y = ge.X, ge.Y
			})
			output.SetModeHandler(func(me client.OutputModeEvent) {
				info.w, info.h = me.Width, me.Height
			})
			outputs = append(outputs, info)
		case "ext_session_lock_manager_v1":
			lockManager = ext_session_lock.NewExtSessionLockManager(display.Context())
			registry.Bind(e.Name, e.Interface, e.Version, lockManager)
		}
	})

	wlRoundTrip(display)
	wlRoundTrip(display)

	if compositor == nil || shm == nil || lockManager == nil || len(outputs) == 0 {
		return
	}

	var keyboard *client.Keyboard
	if seat != nil {
		keyboard, _ = seat.GetKeyboard()
	}

	lock, err := lockManager.Lock()
	if err != nil || lock == nil {
		return
	}

	lockedCh := make(chan struct{}, 1)
	finishedCh := make(chan struct{}, 1)
	lock.SetLockedHandler(func(ext_session_lock.ExtSessionLockLockedEvent) {
		select {
		case lockedCh <- struct{}{}:
		default:
		}
	})
	lock.SetFinishedHandler(func(ext_session_lock.ExtSessionLockFinishedEvent) {
		select {
		case finishedCh <- struct{}{}:
		default:
		}
	})

	escapeCh := make(chan struct{}, 1)
	if keyboard != nil {
		keyboard.SetKeyHandler(func(e client.KeyboardKeyEvent) {
			if e.Key == waylandKeyEscape && client.KeyboardKeyState(e.State) == client.KeyboardKeyStatePressed {
				select {
				case escapeCh <- struct{}{}:
				default:
				}
			}
		})
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

	imgBounds := img.Bounds()

	// The lock surfaces have to be created and painted first: the
	// compositor sends `locked` only once every output is covered by a lock
	// surface with a committed buffer. Waiting for `locked` before creating
	// them deadlocks — the compositor is waiting on the client and the
	// client on the compositor — which is why no overlay appeared at all.
	for _, o := range outputs {
		surface, err := compositor.CreateSurface()
		if err != nil {
			continue
		}
		o.surface = surface

		lockSurface, err := lock.GetLockSurface(surface, o.output)
		if err != nil {
			continue
		}
		o.lockSurface = lockSurface

		info := o
		lockSurface.SetConfigureHandler(func(ce ext_session_lock.ExtSessionLockSurfaceConfigureEvent) {
			lockSurface.AckConfigure(ce.Serial)

			w, h := int32(ce.Width), int32(ce.Height)
			if w == 0 || h == 0 {
				return
			}

			crop := image.Rect(int(info.x), int(info.y), int(info.x)+int(w), int(info.y)+int(h)).Intersect(imgBounds)
			buf := buildWaylandBuffer(shm, tintWhite(img, crop), w, h)
			if buf == nil {
				return
			}

			surface.Attach(buf, 0, 0)
			// Attaching a buffer isn't enough on its own — a commit only
			// presents the parts of it the client marked damaged.
			surface.DamageBuffer(0, 0, w, h)
			surface.Commit()
		})
	}

	// Now the lock can be waited on. `finished` means the compositor
	// refused or dropped it, so there's nothing on screen to keep up;
	// anything else — including a compositor that never gets around to
	// confirming — leaves the surfaces up for their full duration, since
	// they're painted and visible either way.
	locked, aborted := false, false
	select {
	case <-lockedCh:
		locked = true
	case <-finishedCh:
		// The compositor refused or dropped the lock; there's nothing on
		// screen to keep up.
		aborted = true
	case <-time.After(2 * time.Second):
		// Never confirmed, but the surfaces are painted and visible, so
		// they stay up for their full duration anyway.
	}

	if !aborted {
		select {
		case <-escapeCh:
		case <-finishedCh:
			locked = false
		case <-time.After(overlayDuration):
		}
	}

	for _, o := range outputs {
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
