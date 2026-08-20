package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
	"time"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
)

func TestParseOptionsJoinsText(t *testing.T) {
	opts, err := parseOptions([]string{"hello", "there"}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if opts.text != "hello there" {
		t.Fatalf("text = %q, want %q", opts.text, "hello there")
	}
}

func TestParseOptionsRejectsInvalidValues(t *testing.T) {
	tests := [][]string{
		{"-scale", "0", "text"},
		{"-scale", "-1", "text"},
		{"-width", "-1", "text"},
		{"-font-size", "5", "text"},
		{"-text-width", "0", "text"},
		{"-padding", "-1", "text"},
		{"-line-spacing", "0", "text"},
		{"-align", "sideways", "text"},
		{"-text-color", "nope", "text"},
	}
	for _, args := range tests {
		if _, err := parseOptions(args, &bytes.Buffer{}); err == nil {
			t.Errorf("parseOptions(%q) unexpectedly succeeded", args)
		}
	}
}

func TestRenderDimensions(t *testing.T) {
	tests := []struct {
		name string
		opts options
		want image.Rectangle
	}{
		{"scale", func() options { o := defaultOptions(); o.text = "Hi"; o.scale = 0.5; return o }(), image.Rect(0, 0, 1195, 986)},
		{"width", func() options { o := defaultOptions(); o.text = "Hi"; o.width = 800; return o }(), image.Rect(0, 0, 800, 660)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := render(tt.opts)
			if err != nil {
				t.Fatal(err)
			}
			if got.Bounds() != tt.want {
				t.Fatalf("bounds = %v, want %v", got.Bounds(), tt.want)
			}
		})
	}
}

func TestRenderPreservesTransparency(t *testing.T) {
	opts := defaultOptions()
	opts.text = "Hi"
	got, err := render(opts)
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, alpha := got.At(got.Bounds().Max.X-1, got.Bounds().Max.Y-1).RGBA()
	if alpha == 0xffff {
		t.Fatal("bottom-right pixel is opaque; expected background transparency")
	}
}

func TestRenderLongText(t *testing.T) {
	opts := defaultOptions()
	opts.text = strings.Repeat("A very helpful sentence. ", 100)
	if _, err := render(opts); err != nil {
		t.Fatal(err)
	}
}

func TestBubbleGrowsToFitText(t *testing.T) {
	parsedFont, err := truetype.Parse(comicSansFont)
	if err != nil {
		t.Fatal(err)
	}
	dc := gg.NewContext(2390, 1973)

	short := defaultOptions()
	short.text = "Hi"
	shortLayout := measureText(dc, parsedFont, short)
	defer closeFace(shortLayout.face)
	_, _, shortW, shortH := bubbleBounds(shortLayout, short)

	long := defaultOptions()
	long.text = "This message is long enough to make the speech bubble grow in both useful dimensions."
	long.textWidth = 600
	longLayout := measureText(dc, parsedFont, long)
	defer closeFace(longLayout.face)
	_, _, longW, longH := bubbleBounds(longLayout, long)

	if longW <= shortW {
		t.Fatalf("long bubble width = %.0f, want greater than short width %.0f", longW, shortW)
	}
	if longH <= shortH {
		t.Fatalf("long bubble height = %.0f, want greater than short height %.0f", longH, shortH)
	}
}

func TestCharacterLayerRemovesOriginalBubble(t *testing.T) {
	source, _, err := image.Decode(bytes.NewReader(clippyPNG))
	if err != nil {
		t.Fatal(err)
	}
	layer := characterLayer(source)
	_, _, _, alpha := layer.At(100, 100).RGBA()
	if alpha != 0 {
		t.Fatal("original speech bubble remains in character layer")
	}
	_, _, _, alpha = layer.At(2100, 1000).RGBA()
	if alpha == 0 {
		t.Fatal("character was removed from character layer")
	}
	_, _, _, alpha = layer.At(1660, 900).RGBA()
	if alpha == 0 {
		t.Fatal("left side of character was removed from character layer")
	}
}

func TestRunWritesPNGToStdout(t *testing.T) {
	var stdout, stderr bytes.Buffer
	now := func() time.Time { return time.Unix(0, 0) }
	if err := run([]string{"-output", "-", "hello"}, &stdout, &stderr, now); err != nil {
		t.Fatal(err)
	}
	if _, err := png.Decode(bytes.NewReader(stdout.Bytes())); err != nil {
		t.Fatalf("stdout is not a PNG: %v", err)
	}
}

func TestParseHexColor(t *testing.T) {
	got, err := parseHexColor("#12345678")
	if err != nil {
		t.Fatal(err)
	}
	if got != (color.NRGBA{R: 0x12, G: 0x34, B: 0x56, A: 0x78}) {
		t.Fatalf("color = %#v", got)
	}
}

func TestDefaultOutputNameUsesSubsecondPrecision(t *testing.T) {
	a := defaultOutputName(time.Unix(0, 1))
	b := defaultOutputName(time.Unix(0, 2))
	if a == b {
		t.Fatalf("names collide: %q", a)
	}
}
