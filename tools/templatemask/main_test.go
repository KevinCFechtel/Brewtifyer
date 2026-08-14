package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestConvertCreatesAlphaMask(t *testing.T) {
	t.Parallel()

	source := image.NewNRGBA(image.Rect(0, 0, 3, 1))
	source.SetNRGBA(0, 0, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	source.SetNRGBA(1, 0, color.NRGBA{R: 128, G: 128, B: 128, A: 255})
	source.SetNRGBA(2, 0, color.NRGBA{A: 255})

	var input bytes.Buffer
	if err := png.Encode(&input, source); err != nil {
		t.Fatalf("encode source: %v", err)
	}
	var output bytes.Buffer
	if err := convert(&input, &output); err != nil {
		t.Fatalf("convert() error = %v", err)
	}

	converted, err := png.Decode(&output)
	if err != nil {
		t.Fatalf("decode result: %v", err)
	}
	wantAlpha := []uint32{0, 32639, 65535}
	for x, want := range wantAlpha {
		_, _, _, got := converted.At(x, 0).RGBA()
		delta := int64(got) - int64(want)
		if delta < 0 {
			delta = -delta
		}
		if delta > 257 {
			t.Errorf("alpha at x=%d is %d, want approximately %d", x, got, want)
		}
	}
}
