package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"strconv"
	"strings"
)

// PNG layout constants (in pixels). The chart is a simple horizontal bar per
// benchmark with its name on the left and value/speedup on the right.
const (
	pngWidth  = 900
	padX      = 24
	padTop    = 70
	rowH      = 64
	barH      = 26
	labelW    = 170 // reserved column for the benchmark name
	valueW    = 210 // reserved column for the "1.496 S (1.0X)" text
	textScale = 2   // integer upscaling of the 5x7 font
)

var (
	colBG    = color.RGBA{0x16, 0x18, 0x1d, 0xff}
	colText  = color.RGBA{0xe6, 0xe6, 0xe6, 0xff}
	colTrack = color.RGBA{0x2a, 0x2d, 0x35, 0xff}
	// Bars go green (fast) -> amber -> red (slow) by rank, picked per-row.
	barPalette = []color.RGBA{
		{0x4c, 0xc3, 0x8a, 0xff}, // green
		{0xe0, 0xb0, 0x4d, 0xff}, // amber
		{0xd9, 0x6a, 0x5a, 0xff}, // red
		{0x6f, 0x9c, 0xe0, 0xff}, // blue (extras)
	}
)

// writePNG renders the results to a PNG file at path.
func writePNG(path string, rs []result, title string) error {
	base := slowest(rs)
	height := padTop + len(rs)*rowH + padX
	img := image.NewRGBA(image.Rect(0, 0, pngWidth, height))
	fill(img, colBG)

	drawText(img, padX, 24, upper(title), colText, textScale)

	barAreaX := padX + labelW
	barAreaW := pngWidth - barAreaX - valueW - padX

	for i, r := range rs {
		rowY := padTop + i*rowH
		barY := rowY + (rowH-barH)/2

		// Name, vertically centered against the bar.
		drawText(img, padX, barY+(barH-glyphH*textScale)/2, upper(r.name), colText, textScale)

		// Track + filled bar.
		drawRect(img, barAreaX, barY, barAreaW, barH, colTrack)
		filled := max(int(r.nsOp/base*float64(barAreaW)), 2)
		// Fastest bar (last, since sorted slowest-first) gets green; slowest red.
		col := barPalette[rankColor(i, len(rs))]
		drawRect(img, barAreaX, barY, filled, barH, col)

		// Value text to the right of the bar area.
		label := humanTime(r.nsOp) + "  (" + ratio(base/r.nsOp) + "X)"
		drawText(img, barAreaX+barAreaW+12, barY+(barH-glyphH*textScale)/2, upper(label), colText, textScale)
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// rankColor maps a row (0 = slowest) to a palette index: slowest red, fastest
// green, the rest amber/blue.
func rankColor(i, n int) int {
	switch {
	case i == 0:
		return 2 // slowest -> red
	case i == n-1:
		return 0 // fastest -> green
	case i == 1:
		return 1 // amber
	default:
		return 3
	}
}

// ratio formats a speedup like "6.8" or "1.0".
func ratio(x float64) string {
	return strconv.FormatFloat(x, 'f', 1, 64)
}

// upper uppercases s; the bitmap font only carries uppercase letters.
func upper(s string) string {
	return strings.ToUpper(s)
}

// --- tiny drawing primitives -------------------------------------------------

func fill(img *image.RGBA, c color.RGBA) {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			img.SetRGBA(x, y, c)
		}
	}
}

func drawRect(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	for dy := range h {
		for dx := range w {
			img.SetRGBA(x+dx, y+dy, c)
		}
	}
}

// drawText renders s using the 5x7 font, scaled by scale, with the top-left of
// the first glyph at (x, y). Unknown runes render as blanks.
func drawText(img *image.RGBA, x, y int, s string, c color.RGBA, scale int) {
	cx := x
	for _, r := range s {
		glyph, ok := font[r]
		if !ok {
			glyph = font[' ']
		}
		for row := range glyphH {
			bits := glyph[row]
			for col := range glyphW {
				if bits&(1<<(glyphW-1-col)) != 0 {
					drawRect(img, cx+col*scale, y+row*scale, scale, scale, c)
				}
			}
		}
		cx += (glyphW + 1) * scale // one-pixel gap between glyphs
	}
}
