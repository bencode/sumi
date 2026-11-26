package raster

import (
	"image/color"
	"math"
	"testing"

	"github.com/bencode/sumi/mermaid"
	"github.com/bencode/sumi/scene"
)

func TestRenderDrawsWithinTheScene(t *testing.T) {
	fonts, err := LoadFonts()
	if err != nil {
		t.Skip(err)
	}
	s, err := mermaid.Layout("flowchart LR\n  A[代码仓库] -.-> B(Repo)", mermaid.Options{})
	if err != nil {
		t.Fatal(err)
	}
	img, err := Render(s, scene.Light, 2, fonts)
	if err != nil {
		t.Fatal(err)
	}
	if w, h := img.Bounds().Dx(), img.Bounds().Dy(); w != int(math.Ceil(s.Width*2)) || h != int(math.Ceil(s.Height*2)) {
		t.Fatalf("got a %dx%d image, want the scene's %vx%v twice", w, h, s.Width, s.Height)
	}
	bg, _ := scene.Light.Resolve(scene.Use(scene.Background))
	drawn := 0
	for y := range img.Bounds().Dy() {
		for x := range img.Bounds().Dx() {
			if img.RGBAAt(x, y) != (color.RGBA{bg.R, bg.G, bg.B, 255}) {
				drawn++
			}
		}
	}
	if total := img.Bounds().Dx() * img.Bounds().Dy(); drawn == 0 || drawn == total {
		t.Errorf("%d of %d pixels drawn, want some but not all", drawn, total)
	}
}

func TestFontsHaveCJK(t *testing.T) {
	fonts, err := LoadFonts()
	if err != nil {
		t.Skip(err)
	}
	var r renderer
	if _, g := r.glyph(fonts.chain(scene.Font{Size: 16}), '仓'); g == 0 {
		t.Skip("no CJK font on this system")
	}
	for _, f := range []scene.Font{{Bold: true}, {Italic: true}, {Mono: true}} {
		if _, g := r.glyph(fonts.chain(f), '仓'); g == 0 {
			t.Errorf("%+v: no glyph for 仓, though the regular style has one", f)
		}
	}
}
