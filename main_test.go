package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
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

func TestParseOptionsRejectsNonFiniteAndOversizedValues(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{"scale NaN", []string{"-scale", "NaN", "text"}, "-scale"},
		{"scale +Inf", []string{"-scale", "+Inf", "text"}, "-scale"},
		{"scale -Inf", []string{"-scale", "-Inf", "text"}, "-scale"},
		{"font-size NaN", []string{"-font-size", "NaN", "text"}, "-font-size"},
		{"text-x NaN", []string{"-text-x", "NaN", "text"}, "-text-x"},
		{"text-y NaN", []string{"-text-y", "NaN", "text"}, "-text-y"},
		{"text-width NaN", []string{"-text-width", "NaN", "text"}, "-text-width"},
		{"text-height NaN", []string{"-text-height", "NaN", "text"}, "-text-height"},
		{"padding NaN", []string{"-padding", "NaN", "text"}, "-padding"},
		{"line-spacing NaN", []string{"-line-spacing", "NaN", "text"}, "-line-spacing"},
		{"width NaN rejected by parser", []string{"-width", "NaN", "text"}, "-width"},
		{"scale above max", []string{"-scale", "3", "text"}, "-scale"},
		{"width above max", []string{"-width", "6000", "text"}, "-width"},
		{"font-size above max", []string{"-font-size", "5000", "text"}, "-font-size"},
		{"text-width above max", []string{"-text-width", "6000", "text"}, "-text-width"},
		{"text-height above max", []string{"-text-height", "6000", "text"}, "-text-height"},
		{"padding above max", []string{"-padding", "1001", "text"}, "-padding"},
		{"line-spacing above max", []string{"-line-spacing", "11", "text"}, "-line-spacing"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseOptions(tt.args, &bytes.Buffer{})
			if err == nil {
				t.Fatalf("parseOptions(%q) unexpectedly succeeded", tt.args)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("parseOptions(%q) error = %q, want substring %q", tt.args, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestParseOptionsAcceptsExactMaxima(t *testing.T) {
	tests := [][]string{
		{"-scale", "2", "text"},
		{"-width", "5000", "text"},
		{"-font-size", "500", "text"},
		{"-text-width", "5000", "text"},
		{"-text-height", "5000", "text"},
		{"-padding", "1000", "text"},
		{"-line-spacing", "10", "text"},
	}
	for _, args := range tests {
		if _, err := parseOptions(args, &bytes.Buffer{}); err != nil {
			t.Errorf("parseOptions(%q) unexpectedly failed: %v", args, err)
		}
	}
}

func TestParseOptionsHelpDocumentsMaxima(t *testing.T) {
	var stderr bytes.Buffer
	_, err := parseOptions([]string{"-help"}, &stderr)
	if err == nil {
		t.Fatal("parseOptions(-help) unexpectedly succeeded")
	}
	help := stderr.String()
	tests := []string{
		"maximum 2",
		"maximum 5000",
		"maximum 500",
	}
	for _, want := range tests {
		if !strings.Contains(help, want) {
			t.Errorf("help output missing %q:\n%s", want, help)
		}
	}
}

func TestParseOptionsHandlesLeadingDashText(t *testing.T) {
	t.Run("escaped", func(t *testing.T) {
		opts, err := parseOptions([]string{"--", "-1 apples"}, &bytes.Buffer{})
		if err != nil {
			t.Fatal(err)
		}
		if opts.text != "-1 apples" {
			t.Fatalf("text = %q, want %q", opts.text, "-1 apples")
		}
	})
	t.Run("unescaped", func(t *testing.T) {
		_, err := parseOptions([]string{"-1 apples"}, &bytes.Buffer{})
		if err == nil {
			t.Fatal("parseOptions unexpectedly succeeded")
		}
		if !strings.Contains(err.Error(), "--") {
			t.Fatalf("error %q does not suggest using --", err.Error())
		}
	})
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

func TestRenderRejectsInvalidOutputDimensions(t *testing.T) {
	tests := []struct {
		name    string
		width   int
		wantErr string
	}{
		{"too large", 20000, "requested image is too large"},
		{"smaller than one pixel", 1, "requested dimensions are smaller than one pixel"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := defaultOptions()
			opts.text = "Hi"
			opts.width = tt.width
			got, err := render(opts)
			if got != nil {
				t.Fatalf("render() image = %v, want nil", got)
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("render() error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestRenderTextAlignment(t *testing.T) {
	marker := color.NRGBA{R: 10, G: 200, B: 10, A: 255}

	parsedFont, err := truetype.Parse(comicSansFont)
	if err != nil {
		t.Fatal(err)
	}
	dc := gg.NewContext(2390, 1973)

	opts := defaultOptions()
	opts.text = "Hi"
	opts.textColor = marker
	layout := measureText(dc, parsedFont, opts)
	defer closeFace(layout.face)
	bubbleX, bubbleY, bubbleW, bubbleH := bubbleBounds(layout, opts)
	textAreaWidth := bubbleW - 2*opts.padding
	baseX := bubbleX + opts.padding

	expected := map[gg.Align]float64{
		gg.AlignLeft:   baseX,
		gg.AlignCenter: baseX + (textAreaWidth-layout.width)/2,
		gg.AlignRight:  baseX + textAreaWidth - layout.width,
	}

	renderAt := func(align gg.Align) image.Image {
		o := defaultOptions()
		o.text = "Hi"
		o.align = align
		o.textColor = marker
		img, err := render(o)
		if err != nil {
			t.Fatal(err)
		}
		return img
	}

	// firstMarkerX scans only the bubble interior for the first marker-colored
	// pixel, giving the left edge of the rendered glyph for the given image.
	firstMarkerX := func(img image.Image) int {
		minX, minY := int(bubbleX), int(bubbleY)
		maxX, maxY := int(bubbleX+bubbleW), int(bubbleY+bubbleH)
		for x := minX; x < maxX; x++ {
			for y := minY; y < maxY; y++ {
				r, g, bl, _ := img.At(x, y).RGBA()
				if r>>8 < 60 && g>>8 > 150 && bl>>8 < 60 {
					return x
				}
			}
		}
		return -1
	}

	for _, align := range []gg.Align{gg.AlignLeft, gg.AlignCenter, gg.AlignRight} {
		got := firstMarkerX(renderAt(align))
		want := expected[align]
		if math.Abs(float64(got)-want) > 10 {
			t.Errorf("align %v: first marker x = %d, want ~%.1f", align, got, want)
		}
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
