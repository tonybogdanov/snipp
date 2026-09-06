package main

import (
	"image"
	"image/color"
	"time"
)

// overlayDuration is how long the freeze/flash overlay stays up before
// automatically dismissing and saving.
const overlayDuration = 5 * time.Second

// tintWhite crops img to r and blends a 50%-opacity white layer over it,
// returning a new image anchored at (0,0) ready to hand to a platform's
// window-painting code. Compositing is done in software once, up front,
// rather than as two real stacked translucent windows, since genuine
// cross-window alpha blending isn't reliably available everywhere (X11
// needs a compositing manager running; plain GDI/Wayland shm buffers have
// no windowed alpha blending at all) — a single pre-blended image looks
// identical and works everywhere.
func tintWhite(img image.Image, r image.Rectangle) *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	const alpha = 0.5
	for y := 0; y < r.Dy(); y++ {
		for x := 0; x < r.Dx(); x++ {
			sr, sg, sb, _ := img.At(r.Min.X+x, r.Min.Y+y).RGBA()
			r8 := uint8((float64(sr>>8)*(1-alpha) + 255*alpha))
			g8 := uint8((float64(sg>>8)*(1-alpha) + 255*alpha))
			b8 := uint8((float64(sb>>8)*(1-alpha) + 255*alpha))
			out.SetRGBA(x, y, color.RGBA{r8, g8, b8, 255})
		}
	}
	return out
}
