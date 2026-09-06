//go:build linux

package main

import (
	"errors"
	"image"
	"image/color"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

func doScreenshotX11() (string, error) {
	img, err := captureScreenX11()
	if err != nil {
		return "", err
	}
	return saveScreenshot(img)
}

// captureScreenX11 grabs the root window's pixels directly from the X
// server via GetImage. Assumes a 32-bit ZPixmap in BGRX byte order, which
// holds for the TrueColor/DirectColor 24/32-bit depths virtually every
// modern X11 desktop runs.
func captureScreenX11() (image.Image, error) {
	conn, err := xgb.NewConn()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	screen := xproto.Setup(conn).Roots[0]
	w, h := int(screen.WidthInPixels), int(screen.HeightInPixels)

	reply, err := xproto.GetImage(
		conn, xproto.ImageFormatZPixmap, xproto.Drawable(screen.Root),
		0, 0, uint16(w), uint16(h), 0xffffffff,
	).Reply()
	if err != nil {
		return nil, err
	}
	if len(reply.Data) < w*h*4 {
		return nil, errors.New("unexpected image data size from X server")
	}

	data := reply.Data
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := (y*w + x) * 4
			b, g, r := data[i], data[i+1], data[i+2]
			img.SetRGBA(x, y, color.RGBA{r, g, b, 255})
		}
	}
	return img, nil
}
