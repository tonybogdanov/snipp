// Command icongen rescales assets/_source.png — the icon's source of
// truth — into every raster asset the app and its installer embed, since
// tray, window-class and installer APIs need fixed-size raster icons:
//
//	assets/icon.png            32x32 tray icon (linux/other)
//	assets/icon-large.png      256x256, contexts that render the icon large
//	assets/icon.ico            multi-resolution, windows app icon
//	assets/icon-white.png      32x32, the white variant of the tray icon
//	assets/icon-white.ico      multi-resolution, white variant
//	cmd/installer/icon.png     256x256, zenity --window-icon
//	cmd/installer/icon.ico     multi-resolution, installer window class
//
// The white variants are the same mark recolored, not a second drawing, so
// there's still one source of truth to edit. The installer's copies stay
// full-color: they're shown on a dialog, not against a tray background.
//
// Run from the repo root: go run ./cmd/icongen (or bin/icons.ps1)
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

const sourcePath = "assets/_source.png"

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

	ico := encodeSizes(src, false)
	writeFile("assets/icon.ico", ico)
	writeFile("cmd/installer/icon.ico", ico)

	// The white variant, for tray backgrounds the full-color mark disappears
	// against.
	writeFile("assets/icon-white.png", encodePNG(whiten(resize(src, 32))))
	writeFile("assets/icon-white.ico", encodeSizes(src, true))
}

// encodeSizes renders every icoSize and packs them into one .ico, in white
// when asked.
func encodeSizes(src image.Image, white bool) []byte {
	var images []icoImage
	for _, size := range icoSizes {
		img := resize(src, size)
		if white {
			whiten(img)
		}
		images = append(images, icoImage{width: size, height: size, png: encodePNG(img)})
	}

	ico, err := encodeICO(images)
	if err != nil {
		log.Fatal(err)
	}
	return ico
}

// whiten recolors every pixel white in place, keeping the original alpha so
// the mark's shape and its antialiased edges survive. image.RGBA is
// alpha-premultiplied, so an opaque white pixel is R=G=B=A — writing 255s
// into a partly transparent edge pixel would make it brighter than opaque
// white and render as a halo.
func whiten(img *image.RGBA) *image.RGBA {
	for i := 0; i < len(img.Pix); i += 4 {
		a := img.Pix[i+3]
		img.Pix[i+0], img.Pix[i+1], img.Pix[i+2] = a, a, a
	}
	return img
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
