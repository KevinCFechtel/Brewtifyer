package tray

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
)

// iconPNG creates a monochrome template icon so macOS can adapt it to light
// and dark menu bars without requiring a second asset pipeline.
func iconPNG() []byte {
	const size = 32
	icon := image.NewNRGBA(image.Rect(0, 0, size, size))
	black := image.NewUniform(color.NRGBA{A: 255})
	transparent := image.NewUniform(color.NRGBA{})

	// Cup body and base.
	draw.Draw(icon, image.Rect(5, 9, 23, 25), black, image.Point{}, draw.Src)
	draw.Draw(icon, image.Rect(8, 12, 20, 22), transparent, image.Point{}, draw.Src)
	draw.Draw(icon, image.Rect(4, 25, 25, 28), black, image.Point{}, draw.Src)

	// Handle with a transparent center.
	draw.Draw(icon, image.Rect(23, 12, 29, 22), black, image.Point{}, draw.Src)
	draw.Draw(icon, image.Rect(23, 15, 26, 19), transparent, image.Point{}, draw.Src)

	// Two small steam strokes.
	draw.Draw(icon, image.Rect(10, 3, 12, 8), black, image.Point{}, draw.Src)
	draw.Draw(icon, image.Rect(17, 2, 19, 7), black, image.Point{}, draw.Src)

	var buffer bytes.Buffer
	if err := png.Encode(&buffer, icon); err != nil {
		return nil
	}
	return buffer.Bytes()
}
