package main

import (
	"bytes"
	_ "embed"
	"flag"
	"image"
	_ "image/png"
	"log"
	"time"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
)

//go:embed resources/clippy1080.png
var clippyPNG []byte

//go:embed resources/ComicSansMS.ttf
var comicSansFont []byte

func main() {
	scale := flag.Float64("scale", 1.0, "Scale factor (0.5 = half size, 2.0 = double)")
	width := flag.Int("width", 0, "Target width in pixels (maintains aspect ratio, overrides -scale)")
	flag.Parse()

	if flag.NArg() < 1 {
		log.Fatal("Usage: clippy [-scale 0.5] [-width 800] \"Your text here\"")
	}
	text := flag.Arg(0)

	// Load image from embedded bytes
	reader := bytes.NewReader(clippyPNG)
	im, _, err := image.Decode(reader)
	if err != nil {
		log.Fatal(err)
	}
	iw, ih := im.Bounds().Dx(), im.Bounds().Dy()

	// Calculate scale factor
	scaleFactor := *scale
	if *width > 0 {
		scaleFactor = float64(*width) / float64(iw)
	}

	// Calculate output dimensions
	outputW := int(float64(iw) * scaleFactor)
	outputH := int(float64(ih) * scaleFactor)

	// Parse font from embedded bytes
	font, err := truetype.Parse(comicSansFont)
	if err != nil {
		log.Fatal(err)
	}

	// Scale font size proportionally
	fontSize := 80.0 * scaleFactor
	face := truetype.NewFace(font, &truetype.Options{Size: fontSize})

	// Create drawing context
	dc := gg.NewContext(outputW, outputH)
	dc.Scale(scaleFactor, scaleFactor)

	// Clear to transparent background
	dc.SetRGBA(0, 0, 0, 0)
	dc.Clear()
	dc.SetRGB(0, 0, 0)
	dc.DrawImage(im, 0, 0)

	// Draw text
	dc.SetFontFace(face)
	const textWidth = 1200
	dc.DrawStringWrapped(text, 700, 300, 0.5, 0.5, textWidth, 2, gg.AlignLeft)

	// Save output
	tstamp := time.Now().Format("200601020304")
	if err := dc.SavePNG("clippy" + tstamp + ".png"); err != nil {
		log.Fatal(err)
	}
}
