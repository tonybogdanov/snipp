package main

import (
	"bytes"
	"encoding/binary"
)

type icoImage struct {
	width, height int
	png           []byte
}

// encodeICO packs multiple PNG-compressed images (one per size) into a
// single multi-resolution .ico container, which Windows accepts (PNG
// entries have been supported since Vista) and lets it pick whichever
// size best matches the DPI/context it's rendering the icon at, instead
// of stretching a single small bitmap.
func encodeICO(images []icoImage) ([]byte, error) {
	buf := new(bytes.Buffer)

	// ICONDIR
	binary.Write(buf, binary.LittleEndian, uint16(0))                // reserved
	binary.Write(buf, binary.LittleEndian, uint16(1))                // type: icon
	binary.Write(buf, binary.LittleEndian, uint16(len(images))) // image count

	headerSize := 6 + 16*len(images)
	offset := uint32(headerSize)

	// ICONDIRENTRY, one per image
	for _, img := range images {
		w, h := byte(img.width), byte(img.height)
		if img.width >= 256 {
			w = 0
		}
		if img.height >= 256 {
			h = 0
		}
		buf.WriteByte(w)
		buf.WriteByte(h)
		buf.WriteByte(0)                                             // color palette
		buf.WriteByte(0)                                             // reserved
		binary.Write(buf, binary.LittleEndian, uint16(1))            // color planes
		binary.Write(buf, binary.LittleEndian, uint16(32))           // bits per pixel
		binary.Write(buf, binary.LittleEndian, uint32(len(img.png))) // image size
		binary.Write(buf, binary.LittleEndian, offset)
		offset += uint32(len(img.png))
	}

	for _, img := range images {
		buf.Write(img.png)
	}

	return buf.Bytes(), nil
}
