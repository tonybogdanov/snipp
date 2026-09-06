package main

import (
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"time"
)

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

	return filepath.Join(dir, fmt.Sprintf("snipp-%s.png", time.Now().Format("20060102-150405"))), nil
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

// saveScreenshotFile copies an already-captured PNG (e.g. one produced by
// the xdg-desktop-portal) into ~/Pictures/Snipp and returns the new path.
func saveScreenshotFile(srcPath string) (string, error) {
	path, err := newScreenshotPath()
	if err != nil {
		return "", err
	}

	src, err := os.Open(srcPath)
	if err != nil {
		return "", err
	}
	defer src.Close()

	dst, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}
	return path, nil
}
