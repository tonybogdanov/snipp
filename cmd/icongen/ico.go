package main

import (
	"bytes"
	"encoding/binary"
)

// encodeICO wraps a single PNG image in a minimal single-entry .ico
// container, which Windows accepts (PNG-compressed ICO entries have been
// supported since Vista).
func encodeICO(pngData []byte, width, height int) ([]byte, error) {
	buf := new(bytes.Buffer)

	// ICONDIR
	binary.Write(buf, binary.LittleEndian, uint16(0)) // reserved
	binary.Write(buf, binary.LittleEndian, uint16(1)) // type: icon
	binary.Write(buf, binary.LittleEndian, uint16(1)) // image count

	// ICONDIRENTRY
	w, h := byte(width), byte(height)
	if width >= 256 {
		w = 0
	}
	if height >= 256 {
		h = 0
	}
	buf.WriteByte(w)
	buf.WriteByte(h)
	buf.WriteByte(0)                                             // color palette
	buf.WriteByte(0)                                             // reserved
	binary.Write(buf, binary.LittleEndian, uint16(1))            // color planes
	binary.Write(buf, binary.LittleEndian, uint16(32))           // bits per pixel
	binary.Write(buf, binary.LittleEndian, uint32(len(pngData))) // image size
	binary.Write(buf, binary.LittleEndian, uint32(22))           // offset (6 + 16)

	buf.Write(pngData)
	return buf.Bytes(), nil
}
