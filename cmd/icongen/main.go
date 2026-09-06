// Command icongen renders assets/icon.svg's design as raster assets
// (assets/icon.png, assets/icon.ico) for use in the system tray, since
// tray APIs need raster icons rather than SVG.
package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
)

const size = 32

func main() {
	img := render()

	pngBuf := new(bytes.Buffer)
	if err := png.Encode(pngBuf, img); err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile("assets/icon.png", pngBuf.Bytes(), 0644); err != nil {
		log.Fatal(err)
	}

	icoBuf, err := encodeICO(pngBuf.Bytes(), size, size)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile("assets/icon.ico", icoBuf, 0644); err != nil {
		log.Fatal(err)
	}
}

// render draws the crop-corner + scissors mark from assets/icon.svg onto a
// 32x32 monochrome (black-on-transparent) canvas.
func render() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	fg := color.RGBA{0, 0, 0, 255}

	scale := float64(size) / 64.0
	line := func(x1, y1, x2, y2 float64) {
		drawLine(img, x1*scale, y1*scale, x2*scale, y2*scale, fg)
	}

	// crop corners
	line(20, 8, 12, 8)
	line(12, 8, 8, 12)
	line(8, 12, 8, 20)

	line(44, 8, 52, 8)
	line(52, 8, 56, 12)
	line(56, 12, 56, 20)

	line(8, 44, 8, 52)
	line(8, 52, 12, 56)
	line(12, 56, 20, 56)

	line(56, 44, 56, 52)
	line(56, 52, 52, 56)
	line(52, 56, 44, 56)

	// scissors
	drawCircle(img, 22*scale, 20*scale, 4*scale, fg)
	drawCircle(img, 22*scale, 44*scale, 4*scale, fg)
	line(25.5, 22.5, 46, 41.5)
	line(25.5, 41.5, 46, 22.5)

	return img
}

func drawLine(img *image.RGBA, x1, y1, x2, y2 float64, c color.RGBA) {
	dx, dy := x2-x1, y2-y1
	n := 200
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(n)
		x := x1 + dx*t
		y := y1 + dy*t
		setThick(img, x, y, c)
	}
}

func drawCircle(img *image.RGBA, cx, cy, r float64, c color.RGBA) {
	n := 200
	for i := 0; i < n; i++ {
		t := 2 * math.Pi * float64(i) / float64(n)
		x := cx + r*math.Cos(t)
		y := cy + r*math.Sin(t)
		setThick(img, x, y, c)
	}
}

func setThick(img *image.RGBA, x, y float64, c color.RGBA) {
	xi, yi := int(x), int(y)
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			px, py := xi+dx, yi+dy
			if px >= 0 && px < size && py >= 0 && py < size {
				img.Set(px, py, c)
			}
		}
	}
}
