//go:build linux

package main

import (
	"errors"
	"fmt"
	"image"
	_ "image/png"
	"os"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
)

const (
	portalDest      = "org.freedesktop.portal.Desktop"
	portalPath      = "/org/freedesktop/portal/desktop"
	portalScreenIfc = "org.freedesktop.portal.Screenshot"
	portalReqIfc    = "org.freedesktop.portal.Request"
)

// captureScreenWayland asks the compositor's xdg-desktop-portal to take a
// screenshot. This is the only capture path that works uniformly across
// GNOME/Mutter, KDE/KWin and wlroots compositors (Sway, Hyprland, ...),
// since Wayland itself gives clients no access to the framebuffer.
func captureScreenWayland() (image.Image, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if err := conn.AddMatchSignal(
		dbus.WithMatchInterface(portalReqIfc),
		dbus.WithMatchMember("Response"),
	); err != nil {
		return nil, err
	}

	signals := make(chan *dbus.Signal, 1)
	conn.Signal(signals)

	obj := conn.Object(portalDest, dbus.ObjectPath(portalPath))
	options := map[string]dbus.Variant{
		"handle_token": dbus.MakeVariant(fmt.Sprintf("snipp%d", time.Now().UnixNano())),
	}

	var requestPath dbus.ObjectPath
	if err := obj.Call(portalScreenIfc+".Screenshot", 0, "", options).Store(&requestPath); err != nil {
		return nil, fmt.Errorf("portal screenshot request failed: %w", err)
	}

	for sig := range signals {
		if sig.Path != requestPath || sig.Name != portalReqIfc+".Response" {
			continue
		}
		if len(sig.Body) < 2 {
			return nil, errors.New("malformed portal response")
		}
		code, ok := sig.Body[0].(uint32)
		if !ok || code != 0 {
			return nil, fmt.Errorf("screenshot portal declined or failed (code %v)", sig.Body[0])
		}
		results, ok := sig.Body[1].(map[string]dbus.Variant)
		if !ok {
			return nil, errors.New("malformed portal response results")
		}
		uriVal, ok := results["uri"]
		if !ok {
			return nil, errors.New("portal response missing screenshot uri")
		}
		uri, ok := uriVal.Value().(string)
		if !ok {
			return nil, errors.New("portal response uri is not a string")
		}

		path := strings.TrimPrefix(uri, "file://")
		return decodeAndRemove(path)
	}
	return nil, errors.New("portal response channel closed unexpectedly")
}

// decodeAndRemove decodes the portal's own capture file into memory and
// removes it — it only exists as the portal's transient output for this one
// request, and leaving it behind would duplicate every screenshot into
// Pictures itself (the portal writes e.g. ~/Pictures/Screenshot.png).
func decodeAndRemove(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}

	f.Close()
	os.Remove(path)

	return img, nil
}
