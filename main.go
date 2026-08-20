package main

import (
	"bytes"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
)

const (
	baseFontSize = 80
	minFontSize  = 12
	maxPixels    = 100_000_000
	bubbleRadius = 28
	bubbleStroke = 8
	minBubbleW   = 240
)

var bubbleColor = color.NRGBA{R: 255, G: 255, B: 190, A: 255}

//go:embed resources/clippy1080.png
var clippyPNG []byte

//go:embed resources/ComicSansMS.ttf
var comicSansFont []byte

type options struct {
	text        string
	output      string
	scale       float64
	width       int
	fontSize    float64
	textColor   color.Color
	align       gg.Align
	textX       float64
	textY       float64
	textWidth   float64
	textHeight  float64
	padding     float64
	lineSpacing float64
}

func defaultOptions() options {
	return options{
		scale:       1,
		fontSize:    baseFontSize,
		textColor:   color.Black,
		align:       gg.AlignLeft,
		textX:       100,
		textY:       100,
		textWidth:   1200,
		textHeight:  400,
		padding:     40,
		lineSpacing: 2,
	}
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr, time.Now); err != nil {
		fmt.Fprintln(os.Stderr, "clippy:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer, now func() time.Time) error {
	opts, err := parseOptions(args, stderr)
	if err != nil {
		return err
	}

	output, err := render(opts)
	if err != nil {
		return err
	}

	if opts.output == "" {
		opts.output = defaultOutputName(now())
	}
	if opts.output == "-" {
		return png.Encode(stdout, output)
	}

	file, err := os.Create(opts.output)
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	if err := png.Encode(file, output); err != nil {
		_ = file.Close()
		return fmt.Errorf("encode output: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close output: %w", err)
	}
	fmt.Fprintln(stdout, opts.output)
	return nil
}

func parseOptions(args []string, stderr io.Writer) (options, error) {
	opts := defaultOptions()
	var colorValue, alignValue string

	fs := flag.NewFlagSet("clippy", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Float64Var(&opts.scale, "scale", opts.scale, "scale factor (0.5 = half size, 2 = double)")
	fs.IntVar(&opts.width, "width", 0, "target width in pixels; overrides -scale")
	fs.StringVar(&opts.output, "output", "", `output path, or "-" for standard output`)
	fs.Float64Var(&opts.fontSize, "font-size", opts.fontSize, "maximum font size")
	fs.StringVar(&colorValue, "text-color", "#000000", "text color as #RRGGBB or #RRGGBBAA")
	fs.StringVar(&alignValue, "align", "left", "text alignment: left, center, or right")
	fs.Float64Var(&opts.textX, "text-x", opts.textX, "left edge of the maximum text area")
	fs.Float64Var(&opts.textY, "text-y", opts.textY, "top edge of the text")
	fs.Float64Var(&opts.textWidth, "text-width", opts.textWidth, "maximum text width before wrapping")
	fs.Float64Var(&opts.textHeight, "text-height", opts.textHeight, "maximum text height")
	fs.Float64Var(&opts.padding, "padding", opts.padding, "space between the text and bubble edge")
	fs.Float64Var(&opts.lineSpacing, "line-spacing", opts.lineSpacing, "multiplier between text lines")
	fs.Usage = func() {
		fmt.Fprintln(stderr, `Usage: clippy [options] "Your text here"`)
		fmt.Fprintln(stderr, "Options:")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return options{}, err
	}
	if fs.NArg() == 0 {
		fs.Usage()
		return options{}, errors.New("text is required")
	}
	opts.text = strings.Join(fs.Args(), " ")

	opts.textColor, _ = parseHexColor(colorValue)
	if opts.textColor == nil {
		return options{}, fmt.Errorf("invalid -text-color %q", colorValue)
	}
	switch strings.ToLower(alignValue) {
	case "left":
		opts.align = gg.AlignLeft
	case "center":
		opts.align = gg.AlignCenter
	case "right":
		opts.align = gg.AlignRight
	default:
		return options{}, fmt.Errorf("invalid -align %q: use left, center, or right", alignValue)
	}
	if err := validateOptions(opts); err != nil {
		return options{}, err
	}
	return opts, nil
}

func validateOptions(opts options) error {
	switch {
	case opts.scale <= 0:
		return errors.New("-scale must be greater than zero")
	case opts.width < 0:
		return errors.New("-width cannot be negative")
	case opts.fontSize < minFontSize:
		return fmt.Errorf("-font-size must be at least %d", minFontSize)
	case opts.textWidth <= 0 || opts.textHeight <= 0:
		return errors.New("-text-width and -text-height must be greater than zero")
	case opts.padding < 0:
		return errors.New("-padding cannot be negative")
	case opts.lineSpacing <= 0:
		return errors.New("-line-spacing must be greater than zero")
	}
	return nil
}

func render(opts options) (image.Image, error) {
	im, _, err := image.Decode(bytes.NewReader(clippyPNG))
	if err != nil {
		return nil, fmt.Errorf("decode background: %w", err)
	}
	iw, ih := im.Bounds().Dx(), im.Bounds().Dy()

	scaleFactor := opts.scale
	if opts.width > 0 {
		scaleFactor = float64(opts.width) / float64(iw)
	}
	outputW := int(float64(iw) * scaleFactor)
	outputH := int(float64(ih) * scaleFactor)
	if outputW < 1 || outputH < 1 {
		return nil, errors.New("requested dimensions are smaller than one pixel")
	}
	if int64(outputW)*int64(outputH) > maxPixels {
		return nil, fmt.Errorf("requested image is too large (%dx%d)", outputW, outputH)
	}

	font, err := truetype.Parse(comicSansFont)
	if err != nil {
		return nil, fmt.Errorf("parse font: %w", err)
	}

	dc := gg.NewContext(outputW, outputH)
	dc.Scale(scaleFactor, scaleFactor)
	dc.SetRGBA(0, 0, 0, 0)
	dc.Clear()
	drawBubbleAndText(dc, font, opts)
	dc.DrawImage(characterLayer(im), 0, 0)
	return dc.Image(), nil
}

type textLayout struct {
	face   font.Face
	lines  []string
	width  float64
	height float64
}

func bubbleBounds(layout textLayout, opts options) (x, y, width, height float64) {
	width = math.Max(layout.width+2*opts.padding, minBubbleW)
	height = layout.height + 2*opts.padding
	// Keep the bubble's right edge anchored near Clippy as shorter messages
	// collapse, so the pointer does not become needlessly long.
	x = opts.textX + opts.textWidth + opts.padding - width
	y = opts.textY - opts.padding
	return x, y, width, height
}

func measureText(dc *gg.Context, parsedFont *truetype.Font, opts options) textLayout {
	fontSize := opts.fontSize
	for {
		face := truetype.NewFace(parsedFont, &truetype.Options{Size: fontSize})
		dc.SetFontFace(face)
		lines := dc.WordWrap(opts.text, opts.textWidth)
		_, height := dc.MeasureMultilineString(strings.Join(lines, "\n"), opts.lineSpacing)
		if height <= opts.textHeight || fontSize <= minFontSize {
			var width float64
			for _, line := range lines {
				lineWidth, _ := dc.MeasureString(line)
				width = math.Max(width, lineWidth)
			}
			return textLayout{face: face, lines: lines, width: width, height: height}
		}
		closeFace(face)
		fontSize--
	}
}

func drawBubbleAndText(dc *gg.Context, parsedFont *truetype.Font, opts options) {
	layout := measureText(dc, parsedFont, opts)
	defer closeFace(layout.face)

	bubbleX, bubbleY, bubbleW, bubbleH := bubbleBounds(layout, opts)
	bubbleBottom := bubbleY + bubbleH

	pointerBaseX := bubbleX + bubbleW - math.Min(120, bubbleW/3)
	dc.MoveTo(pointerBaseX-45, bubbleBottom-2)
	dc.LineTo(pointerBaseX+45, bubbleBottom-2)
	dc.LineTo(1650, 1100)
	dc.ClosePath()
	dc.SetColor(bubbleColor)
	dc.FillPreserve()
	dc.SetColor(color.Black)
	dc.SetLineWidth(bubbleStroke)
	dc.Stroke()

	dc.DrawRoundedRectangle(bubbleX, bubbleY, bubbleW, bubbleH, bubbleRadius)
	dc.SetColor(bubbleColor)
	dc.FillPreserve()
	dc.SetColor(color.Black)
	dc.SetLineWidth(bubbleStroke)
	dc.Stroke()

	dc.SetFontFace(layout.face)
	dc.SetColor(opts.textColor)
	textAreaWidth := bubbleW - 2*opts.padding
	x := bubbleX + opts.padding
	switch opts.align {
	case gg.AlignCenter:
		x += textAreaWidth / 2
	case gg.AlignRight:
		x += textAreaWidth
	}
	dc.DrawStringWrapped(
		strings.Join(layout.lines, "\n"),
		x,
		opts.textY,
		alignmentAnchor(opts.align),
		0,
		textAreaWidth,
		opts.lineSpacing,
		opts.align,
	)
}

func characterLayer(source image.Image) image.Image {
	layer := image.NewRGBA(source.Bounds())
	bounds := source.Bounds()
	draw.Draw(layer, bounds, source, bounds.Min, draw.Src)

	// The source bubble is a rectangle with a triangular pointer. Clearing its
	// known geometry preserves the adjacent character pixels exactly.
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if (x <= 1600 && y <= 735) ||
				pointInTriangle(x, y, image.Pt(1230, 690), image.Pt(1490, 690), image.Pt(1670, 1140)) {
				layer.SetRGBA(x, y, color.RGBA{})
			}
		}
	}
	return layer
}

func pointInTriangle(x, y int, a, b, c image.Point) bool {
	sign := func(p1, p2, p3 image.Point) int {
		return (p1.X-p3.X)*(p2.Y-p3.Y) - (p2.X-p3.X)*(p1.Y-p3.Y)
	}
	p := image.Pt(x, y)
	d1, d2, d3 := sign(p, a, b), sign(p, b, c), sign(p, c, a)
	hasNegative := d1 < 0 || d2 < 0 || d3 < 0
	hasPositive := d1 > 0 || d2 > 0 || d3 > 0
	return !(hasNegative && hasPositive)
}

func closeFace(face font.Face) {
	if closer, ok := face.(io.Closer); ok {
		_ = closer.Close()
	}
}

func alignmentAnchor(align gg.Align) float64 {
	switch align {
	case gg.AlignCenter:
		return 0.5
	case gg.AlignRight:
		return 1
	default:
		return 0
	}
}

func parseHexColor(value string) (color.Color, error) {
	value = strings.TrimPrefix(value, "#")
	if len(value) != 6 && len(value) != 8 {
		return nil, errors.New("color must contain 6 or 8 hexadecimal digits")
	}
	n, err := strconv.ParseUint(value, 16, 32)
	if err != nil {
		return nil, err
	}
	if len(value) == 6 {
		return color.NRGBA{R: uint8(n >> 16), G: uint8(n >> 8), B: uint8(n), A: 255}, nil
	}
	return color.NRGBA{R: uint8(n >> 24), G: uint8(n >> 16), B: uint8(n >> 8), A: uint8(n)}, nil
}

func defaultOutputName(now time.Time) string {
	return "clippy" + now.Format("20060102T150405.000000000") + ".png"
}
