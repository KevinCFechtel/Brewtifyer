// Command templatemask converts a black-on-white PNG into a macOS template
// image. Black becomes opaque, white becomes transparent, and antialiased gray
// pixels retain proportional alpha coverage.
package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) != 2 {
		return errors.New("usage: templatemask <input.png> <output.png>")
	}

	input, err := os.Open(args[0])
	if err != nil {
		return fmt.Errorf("open input: %w", err)
	}
	defer input.Close()

	output, err := os.Create(args[1])
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}

	if err := convert(input, output); err != nil {
		output.Close()
		return err
	}
	if err := output.Close(); err != nil {
		return fmt.Errorf("close output: %w", err)
	}
	return nil
}

func convert(input io.Reader, output io.Writer) error {
	source, _, err := image.Decode(input)
	if err != nil {
		return fmt.Errorf("decode input: %w", err)
	}

	bounds := source.Bounds()
	result := image.NewNRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, sourceAlpha := source.At(x, y).RGBA()
			luminance := (299*uint64(r) + 587*uint64(g) + 114*uint64(b)) / 1000
			coverage := uint64(sourceAlpha) * (65535 - luminance) / 65535
			result.SetNRGBA(x, y, color.NRGBA{A: uint8((coverage + 128) / 257)})
		}
	}

	encoder := png.Encoder{CompressionLevel: png.BestCompression}
	if err := encoder.Encode(output, result); err != nil {
		return fmt.Errorf("encode template PNG: %w", err)
	}
	return nil
}
