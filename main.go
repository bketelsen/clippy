package main

import (
	"log"
	"os"
	"io/ioutil"
	"path/filepath"
	"time"
	"github.com/fogleman/gg"

	"github.com/gobuffalo/packr/v2"
)

func main() {
	const W = 1200
	const H = 500
	const P = 16

	box := packr.New("myBox", "./resources")
	
	clippy, err := box.Find("clippy1080.png")
	if err != nil {
	  log.Fatal(err)
	}

	cpath := filepath.Join(os.TempDir(), "clippy1080.png")
	err = ioutil.WriteFile(cpath,clippy,0777)
	if err != nil {
	  log.Fatal(err)
	}

	csans, err := box.Find("Comic Sans MS.ttf")
	if err != nil {
	  log.Fatal(err)
	}

	csanspath:= filepath.Join(os.TempDir(), "Comic Sans MS.ttf")
	err = ioutil.WriteFile(csanspath,csans,0777)
	if err != nil {
	  log.Fatal(err)
	}

	im1, err := gg.LoadPNG(cpath)
	if err != nil {
		panic(err)
	}
	iw, ih := im1.Bounds().Dx(), im1.Bounds().Dy()

	dc := gg.NewContext(iw, ih)

	dc.SetRGB(1, 1, 1)
	dc.Clear()
	dc.SetRGB(0, 0, 0)
	dc.DrawImage(im1, 0, 0)
	TEXT := os.Args[1] // "It looks like you are trying to do a thing with some stuff and this should wrap."
	if err := dc.LoadFontFace(csanspath, 80); err != nil {
		panic(err)
	}

	dc.DrawStringWrapped(TEXT, 700, 300, 0.5, 0.5, W, 2, gg.AlignLeft)
	tstamp := time.Now().Format("200601020304")
	dc.SavePNG("clippy"+tstamp+".png")
}
