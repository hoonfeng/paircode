//go:build ignore

package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	dstPath := "cmd/companion/icon.ico"
	sizes := []int{16, 32, 48, 64, 128, 256}

	type imgEntry struct {
		data   []byte
		width  uint8
		height uint8
	}
	var entries []imgEntry

	for _, s := range sizes {
		pngPath := fmt.Sprintf("cmd/companion/icon_%d.png", s)
		data, err := os.ReadFile(pngPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠ 跳过 %dpx: %v\n", s, err)
			continue
		}
		w := uint8(s)
		if s >= 256 {
			w = 0
		}
		entries = append(entries, imgEntry{data: data, width: w, height: w})
	}
	if len(entries) == 0 {
		panic("没有有效的 PNG 文件")
	}

	headerSize := 6 + len(entries)*16
	offset := uint32(headerSize)

	out, _ := os.Create(dstPath)
	defer out.Close()

	binary.Write(out, binary.LittleEndian, uint16(0))
	binary.Write(out, binary.LittleEndian, uint16(1))
	binary.Write(out, binary.LittleEndian, uint16(len(entries)))

	type dirEnt struct {
		off uint32
		len uint32
	}
	var dirs []dirEnt
	for _, e := range entries {
		dirs = append(dirs, dirEnt{off: offset, len: uint32(len(e.data))})
		offset += uint32(len(e.data))
	}
	for i, e := range entries {
		out.Write([]byte{e.width, e.height, 0, 0})
		binary.Write(out, binary.LittleEndian, uint16(1))
		binary.Write(out, binary.LittleEndian, uint16(32))
		binary.Write(out, binary.LittleEndian, dirs[i].len)
		binary.Write(out, binary.LittleEndian, dirs[i].off)
	}
	for _, e := range entries {
		out.Write(e.data)
	}

	abs, _ := filepath.Abs(dstPath)
	fi, _ := out.Stat()
	fmt.Printf("✅ 生成图标: %s (%d 种尺寸, %d 字节)\n", abs, len(entries), fi.Size())

	for _, s := range sizes {
		os.Remove(fmt.Sprintf("cmd/companion/icon_%d.png", s))
	}
	os.Remove("cmd/companion/icon_temp.png")
}
