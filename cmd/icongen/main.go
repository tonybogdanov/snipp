// Command icongen rescales assets/source.png — the icon's source of
// truth — into every raster asset the app and its installer embed, since
// tray, window-class and installer APIs need fixed-size raster icons:
//
//	assets/icon.png            32x32 tray icon (linux/other)
//	assets/icon-large.png      256x256, contexts that render the icon large
//	assets/icon.ico            multi-resolution, windows app icon
//	cmd/installer/icon.png     256x256, zenity --window-icon
//	cmd/installer/icon.ico     multi-resolution, installer window class
//
// Run from the repo root: go run ./cmd/icongen
package main

import (
	"bytes"
	"image"
	"image/png"
	"log"
	"os"

	"golang.org/x/image/draw"
)

// icoSizes covers the tray (small), taskbar/title bar at 100-250% DPI
// scaling, and Explorer/shell (large) presentations.
var icoSizes = []int{16, 24, 32, 48, 64, 128, 256}

const sourcePath = "assets/source.png"

func main() {
	src := loadSource()

	// 256x256 PNG for contexts that render the icon large — the linux
	// installer's dialogs/dock entry — so it isn't a blurry stretch of the
	// 32x32 tray bitmap.
	largePNG := encodePNG(resize(src, 256))
	writeFile("assets/icon-large.png", largePNG)
	writeFile("cmd/installer/icon.png", largePNG)

	// 32x32 PNG stays the tray icon asset (trays render small regardless
	// of DPI, no need for a multi-res image there).
	writeFile("assets/icon.png", encodePNG(resize(src, 32)))

	var images []icoImage
	for _, size := range icoSizes {
		images = append(images, icoImage{width: size, height: size, png: encodePNG(resize(src, size))})
	}

	ico, err := encodeICO(images)
	if err != nil {
		log.Fatal(err)
	}
	writeFile("assets/icon.ico", ico)
	writeFile("cmd/installer/icon.ico", ico)
}

func loadSource() image.Image {
	f, err := os.Open(sourcePath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	src, err := png.Decode(f)
	if err != nil {
		log.Fatal(err)
	}
	return src
}

// resize scales the source down onto a size x size transparent canvas.
// CatmullRom keeps the mark's thin outline readable at tray sizes, where
// a box filter would smear it; the source is square, so no letterboxing
// is needed.
func resize(src image.Image, size int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

func encodePNG(img image.Image) []byte {
	buf := new(bytes.Buffer)
	if err := png.Encode(buf, img); err != nil {
		log.Fatal(err)
	}
	return buf.Bytes()
}

func writeFile(path string, data []byte) {
	if err := os.WriteFile(path, data, 0644); err != nil {
		log.Fatal(err)
	}
}
