package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"time"
)

// doScreenshot captures the screen as fast as possible, then freezes it
// behind a tinted overlay for a few seconds (so the user has visible
// confirmation a screenshot was taken and time to see the result) before
// saving. The overlay is purely presentational: what gets saved is the
// original capture, not whatever the overlay painted.
func doScreenshot() (string, error) {
	img, err := captureScreen()
	if err != nil {
		return "", err
	}

	showFreezeOverlay(img)

	return saveScreenshot(img)
}

// newScreenshotPath returns a fresh, unused path under ~/Pictures/Snipp
// named by the current time, creating that directory if needed.
func newScreenshotPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(home, "Pictures", "Snipp")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	return filepath.Join(dir, fmt.Sprintf("%s.png", time.Now().Format("2006-01-02_15-04-05"))), nil
}

// saveScreenshot encodes img as PNG under ~/Pictures/Snipp and returns the
// path it was written to.
func saveScreenshot(img image.Image) (string, error) {
	path, err := newScreenshotPath()
	if err != nil {
		return "", err
	}

	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		return "", err
	}
	return path, nil
}
