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

// icoSizes covers the tray (small), taskbar/title bar at 100-250% DPI
// scaling, and Explorer/shell (large) presentations.
var icoSizes = []int{16, 24, 32, 48, 64, 128, 256}

var variants = []struct {
	suffix string
	fg     color.RGBA
}{
	{"", color.RGBA{0, 0, 0, 255}},          // assets/icon.{png,ico}: black, for light trays
	{"-white", color.RGBA{255, 255, 255, 255}}, // assets/icon-white.{png,ico}: white, for dark trays
}

func main() {
	for _, v := range variants {
		// 32x32 PNG stays the tray icon asset (trays render small
		// regardless of DPI, no need for a multi-res image there).
		trayPNG := new(bytes.Buffer)
		if err := png.Encode(trayPNG, render(32, v.fg)); err != nil {
			log.Fatal(err)
		}
		if err := os.WriteFile("assets/icon"+v.suffix+".png", trayPNG.Bytes(), 0644); err != nil {
			log.Fatal(err)
		}

		var images []icoImage
		for _, size := range icoSizes {
			buf := new(bytes.Buffer)
			if err := png.Encode(buf, render(size, v.fg)); err != nil {
				log.Fatal(err)
			}
			images = append(images, icoImage{width: size, height: size, png: buf.Bytes()})
		}

		icoBuf, err := encodeICO(images)
		if err != nil {
			log.Fatal(err)
		}
		if err := os.WriteFile("assets/icon"+v.suffix+".ico", icoBuf, 0644); err != nil {
			log.Fatal(err)
		}
	}
}

// render draws the crop-corner + scissors mark from assets/icon.svg onto a
// size x size monochrome (fg-on-transparent) canvas. Stroke thickness
// scales with size so larger icons (taskbar/shell at high DPI) don't look
// like a stretched, blurry version of the tray-sized bitmap.
func render(size int, fg color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	scale := float64(size) / 64.0
	radius := size / 32
	if radius < 1 {
		radius = 1
	}

	line := func(x1, y1, x2, y2 float64) {
		drawLine(img, x1*scale, y1*scale, x2*scale, y2*scale, fg, size, radius)
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
	drawCircle(img, 22*scale, 20*scale, 4*scale, fg, size, radius)
	drawCircle(img, 22*scale, 44*scale, 4*scale, fg, size, radius)
	line(25.5, 22.5, 46, 41.5)
	line(25.5, 41.5, 46, 22.5)

	return img
}

func drawLine(img *image.RGBA, x1, y1, x2, y2 float64, c color.RGBA, size, radius int) {
	dx, dy := x2-x1, y2-y1
	n := 200
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(n)
		x := x1 + dx*t
		y := y1 + dy*t
		setThick(img, x, y, c, size, radius)
	}
}

func drawCircle(img *image.RGBA, cx, cy, r float64, c color.RGBA, size, radius int) {
	n := 200
	for i := 0; i < n; i++ {
		t := 2 * math.Pi * float64(i) / float64(n)
		x := cx + r*math.Cos(t)
		y := cy + r*math.Sin(t)
		setThick(img, x, y, c, size, radius)
	}
}

func setThick(img *image.RGBA, x, y float64, c color.RGBA, size, radius int) {
	xi, yi := int(x), int(y)
	for dx := -radius; dx <= radius; dx++ {
		for dy := -radius; dy <= radius; dy++ {
			px, py := xi+dx, yi+dy
			if px >= 0 && px < size && py >= 0 && py < size {
				img.Set(px, py, c)
			}
		}
	}
}
